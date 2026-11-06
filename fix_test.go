package gosvg

import (
	"image"
	"testing"
)

// 以下回归用例针对审计修复的 bug：在修复前的字节上应当 FAIL，
// 修复后 PASS（证明用例真的咬得住）。

// textMaxAlpha 返回图像中所有非透明像素的最大 alpha（0..255）。
func textMaxAlpha(img *image.NRGBA) int {
	m := 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if int(a>>8) > m {
				m = int(a >> 8)
			}
		}
	}
	return m
}

// hasRedPixel 返回图像中是否存在"红而暗绿暗蓝"的像素。
func hasRedPixel(img *image.NRGBA) bool {
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			R, G, B := r>>8, g>>8, b>>8
			if R > 150 && G < 90 && B < 90 {
				return true
			}
		}
	}
	return false
}

// TestTextOpacitySingle 验证 text 的 opacity 不会被重复乘 ctx.opacity。
// 修复前：opacity*fillOpacity*ctx.opacity → 0.5*1*0.5=0.25（alpha≈64）；
// 修复后：opacity*fillOpacity → 0.5（alpha≈128）。
func TestTextOpacitySingle(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100" opacity="0.5"><text x="10" y="60" font-size="48" fill="black">X</text></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	m := textMaxAlpha(img)
	if m == 0 {
		t.Fatal("no text rendered")
	}
	if m < 110 {
		t.Errorf("text max alpha = %d, want ~128 (opacity 0.5 applied once); got double-applied opacity", m)
	}
}

// TestTextFillInheritance 验证 text 从 <g fill=...> 继承 fill。
// 修复前：renderText 只查 n.attr("fill")，继承失效 → 黑色；修复后：红色。
func TestTextFillInheritance(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><g fill="red"><text x="10" y="60" font-size="48">X</text></g></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !hasRedPixel(img) {
		t.Error("text did not inherit fill=red from parent <g>; rendered black")
	}
}

// TestShapeElementTransform 验证基本形状元素自身的 transform 属性生效。
// 修复前：renderShape 不处理 transform → rect 仍在原点。
func TestShapeElementTransform(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><rect x="0" y="0" width="10" height="10" fill="red" transform="translate(10,10)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if _, _, _, a := img.At(15, 15).RGBA(); a == 0 {
		t.Error("translated rect missing at (15,15)")
	}
	if _, _, _, a := img.At(5, 5).RGBA(); a != 0 {
		t.Error("rect not translated (still at origin)")
	}
}

// TestShapeElementOpacity 验证基本形状元素自身的 opacity 属性生效。
func TestShapeElementOpacity(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="red" opacity="0.5"/></svg>`)
	img, err := Render(data, 10)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	_, _, _, a := img.At(5, 5).RGBA()
	got := a >> 8
	if got < 110 || got > 145 {
		t.Errorf("rect opacity alpha = %d, want ~128 (opacity 0.5)", got)
	}
}

// TestTokenizeScientificNotation 验证科学计数法（含负/正指数）不被错误切分。
// 修复前："1e-5" 被拆成 "1e" + "-5"，parseFloat("1e") 失败。
func TestTokenizeScientificNotation(t *testing.T) {
	tokens := tokenizePath("M0 0 L1e-5 0 L1e+5 0")
	joined := ""
	for _, tok := range tokens {
		joined += tok + "|"
	}
	// 关键 token "1e-5" 和 "1e+5" 必须作为整体存在。
	if !containsToken(tokens, "1e-5") {
		t.Errorf("tokenize lost '1e-5', got tokens: %v", tokens)
	}
	if !containsToken(tokens, "1e+5") {
		t.Errorf("tokenize lost '1e+5', got tokens: %v", tokens)
	}
}

func containsToken(tokens []string, want string) bool {
	for _, tok := range tokens {
		if tok == want {
			return true
		}
	}
	return false
}

// TestInlineStyle 验证 CSS 内联样式 style="fill:..." 生效。
func TestInlineStyle(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><rect x="0" y="0" width="50" height="50" style="fill:red"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	r, g, b, a := img.At(25, 25).RGBA()
	if a == 0 {
		t.Fatal("rect not rendered")
	}
	if R, G, B := r>>8, g>>8, b>>8; !(R > 150 && G < 90 && B < 90) {
		t.Errorf("style fill:red not applied, got rgba=(%d,%d,%d,%d)", R, G, B, a>>8)
	}
}

// TestInlineStyleOverridesAttr 验证内联 style 覆盖 presentation 属性。
func TestInlineStyleOverridesAttr(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><rect width="50" height="50" fill="blue" style="fill:red"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	r, g, b, _ := img.At(25, 25).RGBA()
	R, G, B := r>>8, g>>8, b>>8
	if !(R > 150 && G < 90 && B < 90) {
		t.Errorf("style should override fill=blue, got rgba=(%d,%d,%d)", R, G, B)
	}
}

// TestParseLengthUnit 验证带单位长度的解析（px 剥离）。
func TestParseLengthUnit(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want float64
	}{
		{"512px", 512},
		{"12pt", 12},
		{"2em", 2},
		{"50%", 50},
		{"3.5", 3.5},
	} {
		got, ok := parseLength(tc.in)
		if !ok || got != tc.want {
			t.Errorf("parseLength(%q) = %v, %v; want %v", tc.in, got, ok, tc.want)
		}
	}
}
