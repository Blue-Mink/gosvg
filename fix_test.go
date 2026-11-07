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

// TestStrokeWidthUnit 验证 stroke-width 带单位（如 "4px"）不被回退到默认 1。
// 修复前：parseFloatOr("4px") 失败回退 1 → 渲染成 10px 宽；
// 修复后：parseLengthOr 剥离单位 → 4 → 40px 宽。
func TestStrokeWidthUnit(t *testing.T) {
	// 竖直线，固定 y=50（线中点）数 x 方向非透明像素 = 线宽。
	render := func(strokeWidth string) int {
		data := []byte(`<svg viewBox="0 0 10 10"><line x1="5" y1="1" x2="5" y2="9" stroke="red" stroke-width="` + strokeWidth + `"/></svg>`)
		img, err := Render(data, 100)
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		n := 0
		for x := 0; x < 100; x++ {
			if _, _, _, a := img.At(x, 50).RGBA(); a > 0 {
				n++
			}
		}
		return n
	}
	withUnit := render("4px")
	noUnit := render("4")
	// 两者应一致（≈40px，viewBox 10 → 100 缩放 10x）。
	if withUnit != noUnit {
		t.Errorf("stroke-width=4px → %d px, stroke-width=4 → %d px; should be equal", withUnit, noUnit)
	}
	if withUnit < 35 || withUnit > 45 {
		t.Errorf("stroke-width=4px → %d px, want ~40", withUnit)
	}
}

// TestTextFillNone 验证 text 的 fill="none" / fill="url(#...)" 不渲染任何像素。
// 修复前：parseColor 失败后 c 保持默认黑色 → 渲染成黑色文字。
func TestTextFillNone(t *testing.T) {
	for _, fill := range []string{"none", "url(#g)"} {
		data := []byte(`<svg viewBox="0 0 100 100"><text x="10" y="50" font-size="40" fill="` + fill + `">ABC</text></svg>`)
		img, err := Render(data, 100)
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		if n := countAlpha(img); n != 0 {
			t.Errorf("text fill=%q rendered %d px, want 0", fill, n)
		}
	}
}

// TestUseCycle 验证 use 循环引用不崩溃（修复前会栈溢出 exit status 2）。
func TestUseCycle(t *testing.T) {
	cases := [][]byte{
		// 自引用
		[]byte(`<svg viewBox="0 0 10 10"><use id="a" href="#a"/></svg>`),
		// 互引
		[]byte(`<svg viewBox="0 0 10 10"><use id="a" href="#b"/><use id="b" href="#a"/><use href="#a"/></svg>`),
	}
	for i, data := range cases {
		if _, err := Render(data, 10); err != nil {
			t.Errorf("case %d: use cycle should not error, got %v", i, err)
		}
	}
}

// TestStrokeMiterLimit 验证 stroke-miterlimit 属性生效。
// 修复前：miterLimit 硬编码 4.0 且语义错误（d <= miterLimit*u，实际等价 miterlimit=2）；
// 修复后：正确解析属性，语义为 SVG 规范的 miterLength/strokeWidth（d <= 2*miterLimit*u）。
func TestStrokeMiterLimit(t *testing.T) {
	// 90° 角：miterLength/strokeWidth = 0.707，故 miterlimit=1 保留尖角、0.5 触发 bevel。
	render := func(ml string) int {
		data := []byte(`<svg viewBox="0 0 100 100"><polyline points="5,50 50,50 50,5" fill="none" stroke="red" stroke-width="20" stroke-linejoin="miter" stroke-miterlimit="` + ml + `"/></svg>`)
		img, err := Render(data, 100)
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		return countAlpha(img)
	}
	bevel := render("0.5")
	miter := render("1")
	if bevel == miter {
		t.Errorf("stroke-miterlimit 未生效：0.5 与 1 渲染结果相同（都是 %d px）", bevel)
	}
	if !(bevel < miter) {
		t.Errorf("miterlimit=0.5 应 bevel（尖角砍平，像素更少），got bevel=%d miter=%d", bevel, miter)
	}
}

// TestMergeStyleMiterLimit 验证 mergeStyle 解析 stroke-miterlimit 与默认值。
func TestMergeStyleMiterLimit(t *testing.T) {
	n := &node{attrs: map[string]string{"stroke-miterlimit": "10"}}
	if s := mergeStyle(defaultShapeStyle(), n); s.miterLimit != 10 {
		t.Errorf("miterLimit = %v, want 10", s.miterLimit)
	}
	if s := mergeStyle(defaultShapeStyle(), &node{}); s.miterLimit != 4 {
		t.Errorf("default miterLimit = %v, want 4", s.miterLimit)
	}
}
