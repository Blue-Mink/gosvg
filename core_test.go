package gosvg

import (
	"image"
	"testing"
)

// countAlpha 统计非透明像素数（测试辅助）。
func countAlpha(img *image.NRGBA) int {
	n := 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
				n++
			}
		}
	}
	return n
}

// TestViewBoxScale 验证 viewBox 缩放（512 viewBox → 256 输出）。
func TestViewBoxScale(t *testing.T) {
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><rect width="512" height="512" fill="red"/></svg>`)
	img, err := Render(data, 256)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if n := countAlpha(img); n != 256*256 {
		t.Errorf("full-bleed rect after 2x downscale: got %d px, want %d", n, 256*256)
	}
}

// TestStyleInheritance 验证 fill/stroke 从 svg 根继承到子元素。
func TestStyleInheritance(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 24 24" fill="none" stroke="#00ff00" stroke-width="2"><circle cx="12" cy="12" r="9"/></svg>`)
	img, err := Render(data, 96)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// 中心应透明（fill=none 生效，不能是默认黑色实心）
	r, g, b, a := img.At(48, 48).RGBA()
	if a != 0 {
		t.Errorf("center should be transparent with fill=none, got a=%d rgba=(%d,%d,%d,%d)", a, r, g, b, a>>8)
	}
}

// TestPathArc 验证 path 的 A（arc）指令能正确解析（四分之一圆弧）。
func TestPathArc(t *testing.T) {
	// M0 0 A90 90 0 0 1 90 0 —— 从 (0,0) 到 (90,0) 的四分之一圆弧
	data := []byte(`<svg viewBox="0 0 100 100"><path d="M0 0 A90 90 0 0 1 90 0" fill="none" stroke="black" stroke-width="6"/></svg>`)
	img, err := Render(data, 200)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if n := countAlpha(img); n == 0 {
		t.Fatal("arc path rendered nothing")
	}
}

// TestTransform 验证 transform 属性（translate + scale）。
func TestTransform(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><g transform="translate(10,10)"><rect width="10" height="10" fill="blue"/></g></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// rect 平移到 (10,10)-(20,20)，中心 (15,15) → 像素 (15,15) 应该是蓝色
	if _, _, _, a := img.At(15, 15).RGBA(); a == 0 {
		t.Error("translated rect missing at (15,15)")
	}
	// 原位置 (5,5) 应该是透明的
	if _, _, _, a := img.At(5, 5).RGBA(); a != 0 {
		t.Error("rect not translated (still at origin)")
	}
}
