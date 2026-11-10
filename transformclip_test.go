package gosvg

import "testing"

// SVG 规范：clip-path / mask 定义在「引用该属性的元素的 user space」，
// 即元素自身 transform 生效之后的坐标系。修复前 gosvg 先算 clip/mask 再应用
// transform，导致 <g transform="scale(2)" clip-path="url(#c)"> 被裁错区域
// （真实语料 flag-icons 国旗 IoU 仅 0.206）。
func TestClipPathAppliesAfterElementTransform(t *testing.T) {
	// viewBox 20×20 渲染成 20×20（scale=1）。
	// g 的 transform=scale(2) → g 的 user space 放大 2 倍。
	// clipPath 的 rect 10×10 在 g 的 user space 里 = 父坐标系 20×20 → 不该裁掉任何内容。
	// 内容 rect 10×10 在 g 的 user space 里 = 父坐标系 20×20 → 铺满。
	svg := `<svg viewBox="0 0 20 20">
  <defs><clipPath id="c"><rect x="0" y="0" width="10" height="10"/></clipPath></defs>
  <g transform="scale(2)" clip-path="url(#c)">
    <rect x="0" y="0" width="10" height="10" fill="red"/>
  </g>
</svg>`
	img, err := Render([]byte(svg), 20)
	if err != nil {
		t.Fatal(err)
	}
	if a := img.NRGBAAt(15, 15).A; a == 0 {
		t.Fatal("clip-path must be evaluated in the post-transform user space; " +
			"bottom-right got clipped away")
	}
	if n := nonTransparent(img); n < 380 {
		t.Fatalf("expected the full 20x20 area filled, got %d/400 non-transparent pixels", n)
	}
}

func TestMaskAppliesAfterElementTransform(t *testing.T) {
	svg := `<svg viewBox="0 0 20 20">
  <defs><mask id="m"><rect x="0" y="0" width="10" height="10" fill="white"/></mask></defs>
  <g transform="scale(2)" mask="url(#m)">
    <rect x="0" y="0" width="10" height="10" fill="red"/>
  </g>
</svg>`
	img, err := Render([]byte(svg), 20)
	if err != nil {
		t.Fatal(err)
	}
	if a := img.NRGBAAt(15, 15).A; a == 0 {
		t.Fatal("mask must be evaluated in the post-transform user space")
	}
}

// 没有 transform 时行为不变（回归保护）。
func TestClipPathWithoutTransformUnchanged(t *testing.T) {
	svg := `<svg viewBox="0 0 10 10">
  <defs><clipPath id="c"><rect x="0" y="0" width="5" height="10"/></clipPath></defs>
  <rect width="10" height="10" fill="red" clip-path="url(#c)"/>
</svg>`
	img, err := Render([]byte(svg), 20)
	if err != nil {
		t.Fatal(err)
	}
	if a := img.NRGBAAt(2, 10).A; a == 0 {
		t.Fatal("left half should be kept")
	}
	if a := img.NRGBAAt(18, 10).A; a != 0 {
		t.Fatalf("right half should be clipped away, got alpha %d", a)
	}
}

// 父级 transform + 元素自身 transform + clip-path 三者叠加时，
// clip 应在「元素自身 transform 之后」的坐标系里。
func TestClipPathWithNestedTransforms(t *testing.T) {
	svg := `<svg viewBox="0 0 40 40">
  <defs><clipPath id="c"><rect x="0" y="0" width="10" height="10"/></clipPath></defs>
  <g transform="translate(0,0)">
    <g transform="scale(4)" clip-path="url(#c)">
      <rect x="0" y="0" width="10" height="10" fill="red"/>
    </g>
  </g>
</svg>`
	img, err := Render([]byte(svg), 40)
	if err != nil {
		t.Fatal(err)
	}
	if a := img.NRGBAAt(35, 35).A; a == 0 {
		t.Fatal("clip 10x10 in the scale(4) user space covers the whole 40x40 viewBox")
	}
}

// TestClipPathPreservesChildTransform 验证 clip-path 分支不会把
// skipNodeTransform 泄漏给子树。修复前 <g clip-path> 下所有子元素的
// transform 都被吞掉（真实语料 B__resizeDiagonal 两条 45° 斜线渲染成
// 底部一条横条，IoU 0.0677）。
func TestClipPathPreservesChildTransform(t *testing.T) {
	svg := `<svg viewBox="0 0 10 10">
  <defs><clipPath id="c"><rect width="10" height="10"/></clipPath></defs>
  <g clip-path="url(#c)">
    <rect x="0" y="4" width="10" height="2" fill="red" transform="rotate(90 5 5)"/>
  </g>
</svg>`
	img, err := Render([]byte(svg), 10)
	if err != nil {
		t.Fatal(err)
	}
	if a := img.NRGBAAt(5, 1).A; a == 0 {
		t.Error("子元素的 rotate 被 clip-path 分支吞掉了")
	}
	if a := img.NRGBAAt(1, 5).A; a != 0 {
		t.Error("rect 本应旋转成竖条，仍是横条")
	}
}
