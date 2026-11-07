package gosvg

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestDefaultImageDecoderDataURI(t *testing.T) {
	var buf bytes.Buffer
	one := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	one.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	if err := png.Encode(&buf, one); err != nil {
		t.Fatal(err)
	}
	href := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	dec := DefaultImageDecoder()
	img, err := dec(href)
	if err != nil {
		t.Fatal(err)
	}
	if got := img.Bounds().Dx(); got != 1 {
		t.Fatalf("decoded width = %d, want 1", got)
	}
}

func TestRenderOptionsImageDecoderIsPerRender(t *testing.T) {
	var buf bytes.Buffer
	one := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	one.SetNRGBA(0, 0, color.NRGBA{R: 0, G: 255, B: 0, A: 255})
	if err := png.Encode(&buf, one); err != nil {
		t.Fatal(err)
	}
	href := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	svg := []byte(`<svg viewBox="0 0 2 2"><image href="` + href + `" x="0" y="0" width="2" height="2"/></svg>`)

	// 不注入解码器：仍然静默跳过（默认行为不变）。
	plain, err := Render(svg, 8)
	if err != nil {
		t.Fatal(err)
	}
	if n := nonTransparent(plain); n != 0 {
		t.Fatalf("without decoder: got %d pixels, want 0", n)
	}

	// 通过 RenderOptions 注入：本次渲染生效。
	img, err := RenderWithOptions(context.Background(), svg, 8, 8,
		RenderOptions{ImageDecoder: DefaultImageDecoder()})
	if err != nil {
		t.Fatal(err)
	}
	if n := nonTransparent(img); n == 0 {
		t.Fatal("with per-render decoder: expected image to be drawn")
	}
	// 不应污染包级变量。
	if ImageDecoder != nil {
		t.Fatal("RenderOptions.ImageDecoder must not modify package-level ImageDecoder")
	}
}

func TestDefaultImageDecoderRejectsNonBase64(t *testing.T) {
	if _, err := DefaultImageDecoder()("data:image/png,rawbytes"); err == nil {
		t.Fatal("expected error for non-base64 data URI")
	}
}
