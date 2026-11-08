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

	// data: URI 由内置解码器处理，无需注入。
	dataImg, err := Render(svg, 8)
	if err != nil {
		t.Fatal(err)
	}
	if n := nonTransparent(dataImg); n == 0 {
		t.Fatal("data: URI should render without injection")
	}

	// 文件路径 href 在未注入解码器时必须跳过（安全边界）。
	pathSVG := []byte(`<svg viewBox="0 0 2 2"><image href="/nonexistent.png" x="0" y="0" width="2" height="2"/></svg>`)
	plain, err := Render(pathSVG, 8)
	if err != nil {
		t.Fatal(err)
	}
	if n := nonTransparent(plain); n != 0 {
		t.Fatalf("file href without decoder: got %d pixels, want 0", n)
	}

	// 通过 RenderOptions 注入：本次渲染生效（即使是文件路径 href）。
	img, err := RenderWithOptions(context.Background(), pathSVG, 8, 8,
		RenderOptions{ImageDecoder: func(string) (image.Image, error) { return one, nil }})
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

// data: URI 内联位图应在未注入任何解码器时也能渲染（不涉及文件/网络 IO）。
func TestImageDataURIDecodedWithoutInjector(t *testing.T) {
	if ImageDecoder != nil {
		t.Skip("package-level ImageDecoder already set by another test")
	}
	var buf bytes.Buffer
	one := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	one.SetNRGBA(0, 0, color.NRGBA{R: 0, G: 0, B: 255, A: 255})
	if err := png.Encode(&buf, one); err != nil {
		t.Fatal(err)
	}
	href := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	svg := []byte(`<svg viewBox="0 0 2 2"><image href="` + href + `" width="2" height="2"/></svg>`)
	img, err := Render(svg, 8)
	if err != nil {
		t.Fatal(err)
	}
	if n := nonTransparent(img); n == 0 {
		t.Fatal("data: URI should be decoded without an injected decoder")
	}
}

// 安全边界：文件路径与远程 URL 不得在未显式注入解码器时被读取。
func TestImageFilePathNeedsExplicitDecoder(t *testing.T) {
	if ImageDecoder != nil {
		t.Skip("package-level ImageDecoder already set by another test")
	}
	svg := []byte(`<svg viewBox="0 0 2 2"><image href="/etc/hostname" width="2" height="2"/></svg>`)
	img, err := Render(svg, 8)
	if err != nil {
		t.Fatal(err)
	}
	if n := nonTransparent(img); n != 0 {
		t.Fatal("local file path must not be read without an explicit decoder")
	}
}
