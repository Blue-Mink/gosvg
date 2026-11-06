package gosvg

import (
	"image"
	"testing"
)

func TestRenderMaskCircle(t *testing.T) {
	// 简单 mask：白色圆 mask，目标是红色矩形，结果应该是红色圆形（四角透明）。
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><mask id="a"><circle cx="256" cy="256" r="256" fill="#fff"/></mask><g mask="url(#a)"><path fill="#ff0000" d="M0 0h512v512H0z"/></g></svg>`)
	img, err := Render(data, 256)
	if err != nil {
		t.Fatalf("render err: %v", err)
	}
	// 四角应透明
	for _, p := range []image.Point{{0, 0}, {255, 0}, {0, 255}, {255, 255}} {
		if _, _, _, a := img.At(p.X, p.Y).RGBA(); a != 0 {
			t.Errorf("corner %v alpha = %v, want 0", p, a)
		}
	}
	// 中心应不透明（红色）
	if _, _, _, a := img.At(128, 128).RGBA(); a == 0 {
		t.Errorf("center alpha = 0, want >0")
	}
	// 统计非透明像素数，应该是圆面积 ≈ π*(128)^2
	n := 0
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
				n++
			}
		}
	}
	t.Logf("non-transparent pixels = %d (circle area ~%.0f)", n, 3.14159*128*128)
	if n < 40000 || n > 55000 {
		t.Errorf("pixel count %d out of expected circle range", n)
	}
}

func TestRenderStroke(t *testing.T) {
	// Lucide 风格：fill=none 继承，stroke 圆环。
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#ef4444" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/></svg>`)
	img, err := Render(data, 256)
	if err != nil {
		t.Fatalf("render err: %v", err)
	}
	// 中心应透明（圆环，不是实心）
	if _, _, _, a := img.At(128, 128).RGBA(); a != 0 {
		t.Errorf("center alpha = %v, want 0 (stroke ring, not filled)", a)
	}
	// 边缘（圆环上）应有内容
	n := 0
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
				n++
			}
		}
	}
	if n == 0 {
		t.Errorf("stroke ring empty")
	}
	t.Logf("stroke pixels = %d", n)
}

func TestDefsIndexMask(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 512 512"><mask id="a"><circle cx="256" cy="256" r="256" fill="#fff"/></mask><g mask="url(#a)"><path fill="#f00" d="M0 0h512v512z"/></g></svg>`)
	doc, err := Parse(data)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if _, ok := doc.defs["a"]; !ok {
		t.Fatalf("defs missing mask 'a', got %v", keysOf(doc.defs))
	}
}

func keysOf(m map[string]*node) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func TestParsePathDataSimple(t *testing.T) {
	p := parsePathData("M0 0h512v512H0z")
	t.Logf("path length = %d, svg=%s", len(p), p.ToSVGPath())
	if len(p) == 0 {
		t.Fatal("empty path")
	}
}

func TestRenderRectFill(t *testing.T) {
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" fill="red"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	n := countAlpha(img)
	t.Logf("rect fill alpha pixels = %d", n)
	if n < 9000 {
		t.Fatalf("rect fill too small: %d", n)
	}
}

func TestRenderCircleShape(t *testing.T) {
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><circle cx="256" cy="256" r="256" fill="#fff"/></svg>`)
	img, err := Render(data, 256)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, y := range []int{0, 64, 128, 192, 255} {
		xs := []int{}
		for x := 0; x < 256; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
				xs = append(xs, x)
			}
		}
		if len(xs) > 0 {
			t.Logf("y=%d: range=[%d,%d] n=%d", y, xs[0], xs[len(xs)-1], len(xs))
		} else {
			t.Logf("y=%d: empty", y)
		}
	}
}
