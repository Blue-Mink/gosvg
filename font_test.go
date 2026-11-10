package gosvg

import (
	"bytes"
	"image"
	"os"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// resetFonts 清空系统字体注册表（测试隔离，同包可访问未导出字段）。
func resetFonts() {
	fontRegistry.Lock()
	fontRegistry.families = map[string]*fontFamily{}
	fontRegistry.Unlock()
}

func dejavuDir() string {
	d := "/usr/share/fonts/truetype/dejavu"
	if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
		return ""
	}
	return d
}

func imgEqual(a, b *image.NRGBA) bool {
	return a.Bounds() == b.Bounds() && bytes.Equal(a.Pix, b.Pix)
}

func TestWeightIsBold(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"", false}, {"normal", false}, {"400", false}, {"500", false},
		{"lighter", false}, {"300", false}, {"bold", true}, {"bolder", true},
		{"700", true}, {"900", true}, {"600", true}, {"Bold", true},
	} {
		if got := weightIsBold(tc.in); got != tc.want {
			t.Errorf("weightIsBold(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestIsItalic(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"", false}, {"normal", false}, {"italic", true}, {"oblique", true},
		{"ITALIC", true}, {"Oblique", true},
	} {
		if got := isItalic(tc.in); got != tc.want {
			t.Errorf("isItalic(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestSplitFamilies(t *testing.T) {
	got := splitFamilies(`"DejaVu Sans", Arial, sans-serif`)
	want := []string{"dejavu sans", "arial", "sans-serif"}
	if len(got) != len(want) {
		t.Fatalf("splitFamilies len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("splitFamilies[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestResolveFontUnregisteredFallback(t *testing.T) {
	resetFonts()
	if f := resolveFont("NoSuchFont-12345", false, false); f != nil {
		t.Errorf("resolveFont for unregistered family = %v, want nil", f)
	}
}

// TestRegisterFaceSelection 验证 family 下按字重/字型择优，未命中逐级回退。
// 用 goregular.TTF 解析两次得到两个不同指针，模拟 regular / bold 两张脸。
func TestRegisterFaceSelection(t *testing.T) {
	resetFonts()
	f1, _ := opentype.Parse(goregular.TTF)
	f2, _ := opentype.Parse(goregular.TTF)
	registerFont("TestFace", "Regular", f1)
	registerFont("TestFace", "Bold", f2)

	if got := lookupFace("testface", false, false); got != f1 {
		t.Errorf("lookupFace(regular) = %p, want %p", got, f1)
	}
	if got := lookupFace("testface", true, false); got != f2 {
		t.Errorf("lookupFace(bold) = %p, want %p", got, f2)
	}
	// 无 italic 面：斜体回退到 regular。
	if got := lookupFace("testface", false, true); got != f1 {
		t.Errorf("lookupFace(italic, no italic face) = %p, want %p (regular fallback)", got, f1)
	}
	// 大小写不敏感。
	if got := lookupFont("TESTFACE"); got == nil {
		t.Error("lookupFont(TESTFACE) = nil, want non-nil")
	}
}

func TestGenericFamilyMapping(t *testing.T) {
	resetFonts()
	if f := resolveFont("sans-serif", false, false); f != nil {
		t.Errorf("resolveFont(sans-serif) with empty registry = %v, want nil", f)
	}
	f1, _ := opentype.Parse(goregular.TTF)
	registerFont("DejaVu Sans", "Book", f1)
	if f := resolveFont("sans-serif", false, false); f != f1 {
		t.Errorf("resolveFont(sans-serif) after registering DejaVu Sans = %p, want %p", f, f1)
	}
}

// TestRenderUsesSystemFont 端到端验证：注册 DejaVu 后 font-family 真正换字体，
// 未注册时回退到内置 Go Regular（字节级一致）。
func TestRenderUsesSystemFont(t *testing.T) {
	d := dejavuDir()
	if d == "" {
		t.Skip("DejaVu fonts not present; skipping system-font render test")
	}
	const text = `<svg viewBox="0 0 100 100"><text x="5" y="70" font-size="64" fill="black" font-family="DejaVu Sans">80</text></svg>`
	const noFamily = `<svg viewBox="0 0 100 100"><text x="5" y="70" font-size="64" fill="black">80</text></svg>`

	resetFonts()
	// AddFontDir 会写入包级 fontRegistry；本测试结束必须复位，否则后续测试
	// （尤其 golden 像素哈希）会随执行顺序拿到不同字体而渲染出不同字节。
	// 这也是 golden 基线能安全纳入带 <text> 样例的前提。
	t.Cleanup(resetFonts)
	imgGo, err := Render([]byte(noFamily), 100)
	if err != nil {
		t.Fatalf("render no-family: %v", err)
	}
	// 未注册 DejaVu：font-family 命中失败，回退 Go Regular，应与 no-family 字节一致。
	imgUnreg, err := Render([]byte(text), 100)
	if err != nil {
		t.Fatalf("render unregistered family: %v", err)
	}
	if !imgEqual(imgGo, imgUnreg) {
		t.Error("unregistered font-family should fall back to built-in Go Regular (byte-identical)")
	}

	if err := AddFontDir(d); err != nil {
		t.Fatalf("AddFontDir: %v", err)
	}
	imgDeja, err := Render([]byte(text), 100)
	if err != nil {
		t.Fatalf("render DejaVu: %v", err)
	}
	if textMaxAlpha(imgDeja) == 0 {
		t.Fatal("no text rendered with DejaVu")
	}
	if imgEqual(imgGo, imgDeja) {
		t.Error("font-family='DejaVu Sans' should switch glyphs (slashed zero vs plain), but rendered identical to Go Regular")
	}
}
