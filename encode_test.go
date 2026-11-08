package gosvg

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func solidNRGBA(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return img
}

func TestEncodeImagePNG(t *testing.T) {
	var buf bytes.Buffer
	if err := EncodeImage(&buf, solidNRGBA(4, 4, color.NRGBA{255, 0, 0, 255}), "png", 0, nil); err != nil {
		t.Fatal(err)
	}
	got, err := png.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds().Dx() != 4 || got.Bounds().Dy() != 4 {
		t.Fatalf("decoded bounds = %v", got.Bounds())
	}
}

func TestEncodeImageDefaultFormatIsPNG(t *testing.T) {
	var buf bytes.Buffer
	if err := EncodeImage(&buf, solidNRGBA(2, 2, color.NRGBA{0, 255, 0, 255}), "", 0, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(&buf); err != nil {
		t.Fatalf("default format should be PNG: %v", err)
	}
}

func TestEncodeImageJPEGFlattensOntoWhite(t *testing.T) {
	// 全透明 + 未指定背景 → JPEG 无 alpha，应合成到白色而不是黑色。
	var buf bytes.Buffer
	if err := EncodeImage(&buf, solidNRGBA(4, 4, color.NRGBA{0, 0, 0, 0}), "jpeg", 90, nil); err != nil {
		t.Fatal(err)
	}
	got, err := jpeg.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := got.At(0, 0).RGBA()
	if r>>8 < 240 || g>>8 < 240 || b>>8 < 240 {
		t.Fatalf("expected near-white, got (%d,%d,%d)", r>>8, g>>8, b>>8)
	}
}

func TestEncodeImageJPEGHonoursBackground(t *testing.T) {
	var buf bytes.Buffer
	bg := color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	if err := EncodeImage(&buf, solidNRGBA(4, 4, color.NRGBA{0, 0, 0, 0}), "jpg", 90, bg); err != nil {
		t.Fatal(err)
	}
	got, err := jpeg.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := got.At(0, 0).RGBA()
	if b>>8 < 200 || r>>8 > 60 || g>>8 > 60 {
		t.Fatalf("expected near-blue background, got (%d,%d,%d)", r>>8, g>>8, b>>8)
	}
}

func TestEncodeImageUnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	err := EncodeImage(&buf, solidNRGBA(2, 2, color.NRGBA{0, 0, 0, 255}), "webp", 0, nil)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestRenderToWriterPNG(t *testing.T) {
	svg := []byte(`<svg viewBox="0 0 4 4"><rect width="4" height="4" fill="red"/></svg>`)
	var buf bytes.Buffer
	if err := RenderToWriter(&buf, svg, 8, 8, "png", DefaultRenderOptions()); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 8 {
		t.Fatalf("width = %d, want 8", img.Bounds().Dx())
	}
}

func TestRenderToWriterCtxCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var buf bytes.Buffer
	err := RenderToWriter(&buf, []byte(`<svg viewBox="0 0 4 4"/>`), 8, 8, "png", RenderOptions{Ctx: ctx})
	if err != context.Canceled {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestFlattenForJPEGBlends(t *testing.T) {
	// 50% 透明红 over 白 → 偏粉
	img := solidNRGBA(2, 2, color.NRGBA{R: 255, G: 0, B: 0, A: 128})
	out := flattenForJPEG(img, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	c := out.RGBAAt(0, 0)
	if c.A != 255 {
		t.Fatalf("output must be opaque, got alpha %d", c.A)
	}
	// 50% 透明红 over 白 = (255, ~127, ~127)
	if c.R < 200 || c.G > 160 || c.G < 90 || c.B > 160 || c.B < 90 {
		t.Fatalf("expected pinkish blend ~(255,127,127), got %+v", c)
	}
}
