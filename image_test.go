package gosvg

import (
	"image"
	"image/color"
	"testing"
)

// TestImage 验证 <image> 元素经 ImageDecoder 解码并渲染到指定矩形。
func TestImage(t *testing.T) {
	orig := ImageDecoder
	defer func() { ImageDecoder = orig }()

	ImageDecoder = func(href string) (image.Image, error) {
		if href != "test" {
			t.Errorf("href 应为 test，got %q", href)
		}
		m := image.NewRGBA(image.Rect(0, 0, 2, 2))
		for i := range m.Pix {
			m.Pix[i] = 255
		}
		m.Set(0, 0, color.RGBA{255, 0, 0, 255})
		return m, nil
	}

	data := []byte(`<svg viewBox="0 0 100 100"><image href="test" x="0" y="0" width="50" height="50"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// image 占左上 50×50，内部应不透明
	if _, _, _, a := img.At(25, 25).RGBA(); a == 0 {
		t.Error("image 应渲染到左上角")
	}
	// 区域外（右下）应透明
	if _, _, _, a := img.At(75, 75).RGBA(); a != 0 {
		t.Error("image 区域外应透明")
	}
}

// TestImageNoDecoder 验证无 ImageDecoder 时 <image> 静默跳过。
func TestImageNoDecoder(t *testing.T) {
	orig := ImageDecoder
	ImageDecoder = nil
	defer func() { ImageDecoder = orig }()

	data := []byte(`<svg viewBox="0 0 100 100"><image href="test" x="0" y="0" width="50" height="50"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if n := countAlpha(img); n != 0 {
		t.Errorf("无 decoder 时 image 应跳过（全透明），got %d 不透明像素", n)
	}
}

// TestImageDefaultSize 验证未指定 width/height 时用位图固有尺寸。
func TestImageDefaultSize(t *testing.T) {
	orig := ImageDecoder
	defer func() { ImageDecoder = orig }()

	ImageDecoder = func(href string) (image.Image, error) {
		m := image.NewRGBA(image.Rect(0, 0, 10, 10))
		for i := range m.Pix {
			m.Pix[i] = 255
		}
		return m, nil
	}

	data := []byte(`<svg viewBox="0 0 100 100"><image href="x" x="0" y="0"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// 固有尺寸 10×10（用户单位，1:1 缩放），占 (0,0)-(10,10)
	if _, _, _, a := img.At(5, 5).RGBA(); a == 0 {
		t.Error("默认尺寸 image 应渲染到 (0,0)-(10,10)")
	}
	if _, _, _, a := img.At(20, 20).RGBA(); a != 0 {
		t.Error("默认尺寸 image 区域外应透明")
	}
}
