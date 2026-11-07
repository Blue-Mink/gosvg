package gosvg

import (
	"math"
	"testing"
)

// TestMeasureTextBasic 验证 MeasureText 返回正宽度。
func TestMeasureTextBasic(t *testing.T) {
	if w := MeasureText("Hello", 20, ""); w <= 0 {
		t.Errorf("MeasureText 应返回正宽度，got %v", w)
	}
}

// TestMeasureTextEmpty 验证空文本返回 0。
func TestMeasureTextEmpty(t *testing.T) {
	if w := MeasureText("", 20, ""); w != 0 {
		t.Errorf("空文本应返回 0，got %v", w)
	}
}

// TestMeasureTextScalesWithSize 验证字号缩放线性。
func TestMeasureTextScalesWithSize(t *testing.T) {
	w20 := MeasureText("Hello", 20, "")
	w40 := MeasureText("Hello", 40, "")
	if w20 <= 0 {
		t.Fatal("基准宽度应为正")
	}
	if ratio := w40 / w20; ratio < 1.9 || ratio > 2.1 {
		t.Errorf("字号翻倍宽度应约翻倍：w20=%v w40=%v ratio=%v", w20, w40, ratio)
	}
}

// TestMeasureTextMatchesRender 验证测量宽度与实际渲染边界一致。
func TestMeasureTextMatchesRender(t *testing.T) {
	const text = "Hello"
	const fontSize = 40.0
	data := []byte(`<svg viewBox="0 0 200 100"><text x="0" y="60" font-size="40">` + text + `</text></svg>`)
	img, err := Render(data, 200)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	maxX := -1
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
				if x > maxX {
					maxX = x
				}
			}
		}
	}
	if maxX < 0 {
		t.Fatal("文本未渲染")
	}
	w := MeasureText(text, fontSize, "")
	// 1:1 无缩放，渲染边界 ≈ 测量宽度（允许 2px AA 误差）
	if math.Abs(float64(maxX+1)-w) > 2 {
		t.Errorf("测量宽度 %v 与渲染边界 %d 不一致", w, maxX+1)
	}
}
