package gosvg

import (
	"image"
	"testing"
)

func nonTransparent(img *image.NRGBA) int {
	n := 0
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] > 0 {
			n++
		}
	}
	return n
}

func renderPixels(t *testing.T, svg string) int {
	t.Helper()
	img, err := Render([]byte(svg), 16)
	if err != nil {
		t.Fatal(err)
	}
	return nonTransparent(img)
}

func TestDisplayNone(t *testing.T) {
	if n := renderPixels(t, `<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="red" display="none"/></svg>`); n != 0 {
		t.Fatalf("display=none: got %d non-transparent pixels, want 0", n)
	}
}

func TestDisplayNoneInInlineStyle(t *testing.T) {
	if n := renderPixels(t, `<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="red" style="display:none"/></svg>`); n != 0 {
		t.Fatalf("style display:none: got %d, want 0", n)
	}
}

func TestDisplayNoneFromCSS(t *testing.T) {
	svg := `<svg viewBox="0 0 10 10"><style>.hidden{display:none}</style><rect width="10" height="10" fill="red" class="hidden"/></svg>`
	if n := renderPixels(t, svg); n != 0 {
		t.Fatalf("CSS display:none: got %d, want 0", n)
	}
}

func TestDisplayNoneSkipsSubtree(t *testing.T) {
	if n := renderPixels(t, `<svg viewBox="0 0 10 10"><g display="none"><rect width="10" height="10" fill="red"/></g></svg>`); n != 0 {
		t.Fatalf("display=none on <g>: got %d, want 0", n)
	}
}

func TestVisibilityHidden(t *testing.T) {
	if n := renderPixels(t, `<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="red" visibility="hidden"/></svg>`); n != 0 {
		t.Fatalf("visibility=hidden: got %d, want 0", n)
	}
}

func TestVisibilityHiddenCollapse(t *testing.T) {
	if n := renderPixels(t, `<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="red" visibility="collapse"/></svg>`); n != 0 {
		t.Fatalf("visibility=collapse: got %d, want 0", n)
	}
}

func TestVisibilityVisibleOverridesParent(t *testing.T) {
	svg := `<svg viewBox="0 0 10 10"><g visibility="hidden"><rect width="10" height="10" fill="red" visibility="visible"/></g></svg>`
	if n := renderPixels(t, svg); n != 256 {
		t.Fatalf("child visibility=visible should restore drawing, got %d, want 256", n)
	}
}

func TestVisibilityInheritsHidden(t *testing.T) {
	svg := `<svg viewBox="0 0 10 10"><g visibility="hidden"><rect width="10" height="10" fill="red"/></g></svg>`
	if n := renderPixels(t, svg); n != 0 {
		t.Fatalf("visibility should inherit to children, got %d, want 0", n)
	}
}

func TestDisplayAndVisibilityBaseline(t *testing.T) {
	if n := renderPixels(t, `<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="red"/></svg>`); n != 256 {
		t.Fatalf("baseline: got %d, want 256", n)
	}
}
