package gosvg

import (
	"image/color"
	"testing"
)

// TestParseColorHSL 验证 hsl() 解析（红/绿/蓝/白/黑/灰）。
func TestParseColorHSL(t *testing.T) {
	cases := []struct {
		in   string
		want color.NRGBA
	}{
		{"hsl(0, 100%, 50%)", color.NRGBA{255, 0, 0, 255}},
		{"hsl(120, 100%, 50%)", color.NRGBA{0, 255, 0, 255}},
		{"hsl(240, 100%, 50%)", color.NRGBA{0, 0, 255, 255}},
		{"hsl(0, 0%, 100%)", color.NRGBA{255, 255, 255, 255}},
		{"hsl(0, 0%, 0%)", color.NRGBA{0, 0, 0, 255}},
		{"hsl(0, 0%, 50%)", color.NRGBA{128, 128, 128, 255}},
		// 空格分隔语法
		{"hsl(120 100% 50%)", color.NRGBA{0, 255, 0, 255}},
		// 度单位
		{"hsl(120deg, 100%, 50%)", color.NRGBA{0, 255, 0, 255}},
	}
	for _, tc := range cases {
		got, ok := parseColor(tc.in)
		if !ok {
			t.Errorf("parseColor(%q) 失败", tc.in)
			continue
		}
		d := absInt(int(got.R)-int(tc.want.R)) + absInt(int(got.G)-int(tc.want.G)) + absInt(int(got.B)-int(tc.want.B))
		if d > 3 {
			t.Errorf("parseColor(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestParseColorHSLA 验证 hsla() 的 alpha 分量。
func TestParseColorHSLA(t *testing.T) {
	c, ok := parseColor("hsla(0, 100%, 50%, 0.5)")
	if !ok {
		t.Fatal("hsla 解析失败")
	}
	if c.R != 255 || c.G != 0 || c.B != 0 {
		t.Errorf("hsla 颜色错误: %v", c)
	}
	if c.A < 126 || c.A > 130 {
		t.Errorf("hsla alpha = %d, want ~128", c.A)
	}
	// 斜杠语法
	c2, ok := parseColor("hsl(0 100% 50% / 0.5)")
	if !ok {
		t.Fatal("hsl 斜杠 alpha 解析失败")
	}
	if c2.A < 126 || c2.A > 130 {
		t.Errorf("hsl 斜杠 alpha = %d, want ~128", c2.A)
	}
}

// TestParseColorRGBSpaceSep 验证 rgb() 空格分隔与斜杠 alpha 语法。
func TestParseColorRGBSpaceSep(t *testing.T) {
	c, ok := parseColor("rgb(255 0 0)")
	if !ok || c.R != 255 || c.G != 0 || c.B != 0 || c.A != 255 {
		t.Errorf("rgb(255 0 0) = %v ok=%v, want 红", c, ok)
	}
	c2, ok := parseColor("rgb(255 0 0 / 0.5)")
	if !ok || c2.A < 126 || c2.A > 130 {
		t.Errorf("rgb(255 0 0 / 0.5) = %v ok=%v, want alpha≈128", c2, ok)
	}
	// 逗号分隔 + 第 4 分量（rgba 老语法）
	c3, ok := parseColor("rgba(255, 0, 0, 0.5)")
	if !ok || c3.A < 126 || c3.A > 130 {
		t.Errorf("rgba(255,0,0,0.5) = %v ok=%v, want alpha≈128", c3, ok)
	}
}

// TestRenderHSL 验证 hsl 颜色在渲染链路里生效。
func TestRenderHSL(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="hsl(120, 100%, 50%)"/></svg>`)
	img, err := Render(data, 10)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	r, g, b, a := img.At(5, 5).RGBA()
	if a == 0 {
		t.Fatal("rect 未渲染")
	}
	R, G, B := r>>8, g>>8, b>>8
	if !(G > 200 && R < 60 && B < 60) {
		t.Errorf("fill=hsl(120,100%%,50%%) 应绿色, got rgba=(%d,%d,%d,%d)", R, G, B, a>>8)
	}
}
