package gosvg

import (
	"context"
	"image/color"
	"strings"
	"testing"
)

func TestRenderOptionsMaxDimension(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="red"/></svg>`)
	if _, err := Render(data, DefaultMaxDimension+1); err == nil {
		t.Fatal("expected size limit error for default MaxDimension")
	}
	// 显式放宽上限后应可渲染（用小尺寸验证参数生效，不真的分配超大图）。
	if _, err := RenderWithOptions(context.Background(), data, 16, 16,
		RenderOptions{MaxDimension: 8}); err == nil {
		t.Fatal("expected size limit error for custom MaxDimension=8")
	}
	if _, err := Render(data, 64); err != nil {
		t.Fatalf("normal size should pass: %v", err)
	}
}

func TestRenderOptionsBackground(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 10 10"/>`)
	bg := color.NRGBA{R: 10, G: 20, B: 30, A: 255}
	img, err := RenderWithOptions(context.Background(), data, 8, 8, RenderOptions{Background: bg})
	if err != nil {
		t.Fatal(err)
	}
	if got := img.NRGBAAt(0, 0); got != bg {
		t.Fatalf("background = %v, want %v", got, bg)
	}
	plain, err := Render(data, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got := plain.NRGBAAt(0, 0); got.A != 0 {
		t.Fatalf("default background should be transparent, got %v", got)
	}
}

func TestRenderWithOptionsContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	data := []byte(`<svg viewBox="0 0 10 10"><rect width="10" height="10"/></svg>`)
	if _, err := RenderWithOptions(ctx, data, 16, 16, DefaultRenderOptions()); err != context.Canceled {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	doc, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.RenderWithOptions(ctx, 16, 16, DefaultRenderOptions()); err != context.Canceled {
		t.Fatalf("Document err = %v, want context.Canceled", err)
	}
}

func TestDiagnostics(t *testing.T) {
	svg := []byte(`<svg viewBox="0 0 10 10">
  <filter id="f"><feGaussianBlur/></filter>
  <image href="missing.png" x="0" y="0" width="5" height="5"/>
  <rect width="5" height="5" fill="red" mask="url(#nope)"/>
  <use href="#missing"/>
</svg>`)
	doc, err := Parse(svg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.Render(32, 32); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(doc.Diagnostics(), "\n")
	for _, want := range []string{
		"unsupported element skipped: filter",
		"ImageDecoder not set",
		"mask reference not found",
		"use reference not found",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing diagnostic %q; got:\n%s", want, joined)
		}
	}
	clean, err := Parse([]byte(`<svg viewBox="0 0 4 4"><rect width="4" height="4" fill="blue"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := clean.Render(8, 8); err != nil {
		t.Fatal(err)
	}
	if d := clean.Diagnostics(); len(d) != 0 {
		t.Fatalf("clean document diagnostics = %v, want empty", d)
	}
}

func TestDiagnosticsDeduplicates(t *testing.T) {
	doc, err := Parse([]byte(`<svg viewBox="0 0 4 4"><filter id="a"/><filter id="b"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.Render(8, 8); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, d := range doc.Diagnostics() {
		if d == "unsupported element skipped: filter" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("diagnostic should be deduplicated, got %d", n)
	}
}
