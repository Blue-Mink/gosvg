package gosvg

import "testing"

// TestClipPath 验证 clip-path="url(#id)" 基本裁剪（userSpaceOnUse 默认）。
func TestClipPath(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><clipPath id="c"><circle cx="50" cy="50" r="40"/></clipPath><rect width="100" height="100" fill="#f00" clip-path="url(#c)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if _, _, _, a := img.At(50, 50).RGBA(); a == 0 {
		t.Error("clip 圆内应可见")
	}
	if _, _, _, a := img.At(5, 5).RGBA(); a != 0 {
		t.Error("clip 圆外应透明")
	}
}

// TestClipPathUnitsObjectBoundingBox 验证 clipPathUnits=objectBoundingBox
// 的归一化坐标映射。
func TestClipPathUnitsObjectBoundingBox(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><clipPath id="c" clipPathUnits="objectBoundingBox"><rect x="0" y="0" width="0.5" height="1"/></clipPath><rect x="20" y="20" width="60" height="60" fill="#f00" clip-path="url(#c)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// clip 区域 = bbox 左半 → 左半可见
	if _, _, _, a := img.At(30, 50).RGBA(); a == 0 {
		t.Error("clipPathUnits=objectBoundingBox 左半应可见")
	}
	if _, _, _, a := img.At(70, 50).RGBA(); a != 0 {
		t.Error("clipPathUnits=objectBoundingBox 右半应透明")
	}
}

// TestClipPathUnion 验证 clipPath 多个子元素的并集裁剪。
func TestClipPathUnion(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><clipPath id="c"><rect x="0" y="0" width="50" height="100"/><rect x="50" y="0" width="50" height="50"/></clipPath><rect width="100" height="100" fill="#f00" clip-path="url(#c)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// 左半(25,50) 可见
	if _, _, _, a := img.At(25, 50).RGBA(); a == 0 {
		t.Error("clip 左半应可见")
	}
	// 右上象限(75,25) 可见
	if _, _, _, a := img.At(75, 25).RGBA(); a == 0 {
		t.Error("clip 右上象限应可见")
	}
	// 右下象限(75,75) 透明（不在 clip union 内）
	if _, _, _, a := img.At(75, 75).RGBA(); a != 0 {
		t.Error("clip 右下象限应透明")
	}
}
