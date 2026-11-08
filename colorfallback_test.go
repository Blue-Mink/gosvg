package gosvg

import "testing"

// 无效颜色声明应按 CSS 语义被忽略、保持继承值，而不是让图形整体消失。
// 真实语料实测：Material Symbols 导出的 fill="000000"（缺 #）修复前渲染为全透明。
func TestInvalidFillFallsBackToInherited(t *testing.T) {
	if n := renderPixels(t, `<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="000000"/></svg>`); n != 256 {
		t.Fatalf(`fill="000000": got %d pixels, want 256 (fallback to black)`, n)
	}
	if n := renderPixels(t, `<svg viewBox="0 0 10 10" fill="000000"><rect width="10" height="10"/></svg>`); n != 256 {
		t.Fatalf("invalid fill on root: got %d, want 256", n)
	}
}

func TestValidNoneStillMeansNoFill(t *testing.T) {
	if n := renderPixels(t, `<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="none"/></svg>`); n != 0 {
		t.Fatalf(`fill="none": got %d, want 0`, n)
	}
}

func TestInvalidStrokeIgnoredButValidWorks(t *testing.T) {
	if n := renderPixels(t, `<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="none" stroke="000000" stroke-width="2"/></svg>`); n != 0 {
		t.Fatalf("invalid stroke should be ignored: got %d, want 0", n)
	}
	if n := renderPixels(t, `<svg viewBox="0 0 10 10"><rect x="1" y="1" width="8" height="8" fill="none" stroke="#f00" stroke-width="2"/></svg>`); n == 0 {
		t.Fatal("valid stroke should still draw")
	}
}

func TestIsValidPaint(t *testing.T) {
	for _, v := range []string{"red", "#fff", "#ff0000", "rgb(1,2,3)", "hsl(120 50% 50%)",
		"none", "transparent", "currentColor", "url(#g)", "  url(#g)  "} {
		if !isValidPaint(v) {
			t.Errorf("isValidPaint(%q) = false, want true", v)
		}
	}
	for _, v := range []string{"", "   ", "000000", "notacolor", "#gggggg", "url(#g", "12px"} {
		if isValidPaint(v) {
			t.Errorf("isValidPaint(%q) = true, want false", v)
		}
	}
}

func TestParseColorRejectsInvalidHex(t *testing.T) {
	for _, s := range []string{"#gggggg", "#fff!", "#12345g", "#", "#zzz"} {
		if c, ok := parseColor(s); ok {
			t.Errorf("parseColor(%q) = %v, true; want false", s, c)
		}
	}
	for _, s := range []string{"#fff", "#ffff", "#ff0000", "#ff000080", "#ABCDEF"} {
		if _, ok := parseColor(s); !ok {
			t.Errorf("parseColor(%q) = false; want true", s)
		}
	}
}
