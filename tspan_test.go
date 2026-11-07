package gosvg

import (
	"image"
	"testing"
)

// countPx 统计图片中满足 pred 的像素数。
func countPx(img *image.NRGBA, pred func(r, g, b, a uint8) bool) int {
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.NRGBAAt(x, y)
			if pred(c.R, c.G, c.B, c.A) {
				n++
			}
		}
	}
	return n
}

func redPx(img *image.NRGBA) int {
	return countPx(img, func(r, g, b, a uint8) bool { return a > 0 && r > 200 && g < 100 && b < 100 })
}

func blackPx(img *image.NRGBA) int {
	return countPx(img, func(r, g, b, a uint8) bool { return a > 0 && r < 60 && g < 60 && b < 60 })
}

// TestTSpanOnly 验证仅含 tspan 的文本被渲染（之前 tspan 被忽略）。
func TestTSpanOnly(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><text x="10" y="50" font-size="40"><tspan fill="#ff0000">X</tspan></text></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if redPx(img) == 0 {
		t.Error("tspan 的文本应被渲染（红色）")
	}
}

// TestTSpanBasic 验证直接文本 + tspan 共存，各自 fill 生效。
func TestTSpanBasic(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 200 100"><text x="10" y="50" font-size="40">Hi <tspan fill="#ff0000">Bye</tspan></text></svg>`)
	img, err := Render(data, 200)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if blackPx(img) == 0 {
		t.Error("直接文本 'Hi ' 应渲染（默认黑）")
	}
	if redPx(img) == 0 {
		t.Error("tspan 'Bye' 应渲染（红色）")
	}
}

// TestTSpanInheritFill 验证 tspan 未设 fill 时继承 text 的 fill。
func TestTSpanInheritFill(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><text x="10" y="50" font-size="40" fill="#ff0000"><tspan>X</tspan></text></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if redPx(img) == 0 {
		t.Error("tspan 应继承 text 的红色 fill")
	}
}

// TestTSpanNested 验证嵌套 tspan 的 fill 覆盖。
func TestTSpanNested(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 200 100"><text x="10" y="50" font-size="40"><tspan>A<tspan fill="#ff0000">B</tspan></tspan></text></svg>`)
	img, err := Render(data, 200)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if blackPx(img) == 0 {
		t.Error("外层 tspan 'A' 应渲染（默认黑）")
	}
	if redPx(img) == 0 {
		t.Error("内层 tspan 'B' 应渲染（红色）")
	}
}

// TestTSpanXOffset 验证 tspan 的绝对 x 使文本落到指定位置（与起始 x 分离）。
func TestTSpanXOffset(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 200 100"><text x="10" y="50" font-size="40"><tspan x="120" fill="#ff0000">X</tspan></text></svg>`)
	img, err := Render(data, 200)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// 红色像素应集中在右侧（x>=100），左侧（x<100）无红色。
	left, right := 0, 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.NRGBAAt(x, y)
			if c.A > 0 && c.R > 200 && c.G < 100 && c.B < 100 {
				if x < 100 {
					left++
				} else {
					right++
				}
			}
		}
	}
	if right == 0 {
		t.Error("tspan x=120 的文本应渲染在右侧")
	}
	if left > right {
		t.Errorf("tspan x=120 应靠右：left=%d right=%d", left, right)
	}
}
