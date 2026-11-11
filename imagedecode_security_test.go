package gosvg

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// makeDataURI 生成 1×1 PNG 的 data URI。
func makeDataURI(t *testing.T, c color.NRGBA) string {
	t.Helper()
	var buf bytes.Buffer
	one := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	one.SetNRGBA(0, 0, c)
	if err := png.Encode(&buf, one); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// TestDefaultImageDecoderIsDataURIOnly 是 v0.2.4 安全收紧的核心回归：
// DefaultImageDecoder 绝不能再读取本地文件。
func TestDefaultImageDecoderIsDataURIOnly(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "pic.png")
	var buf bytes.Buffer
	one := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	one.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	if err := png.Encode(&buf, one); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	dec := DefaultImageDecoder()
	if _, err := dec(file); err == nil {
		t.Fatal("DefaultImageDecoder 读取了本地文件 —— 安全边界被破坏")
	}
	if _, err := dec("https://example.com/x.png"); err == nil {
		t.Fatal("DefaultImageDecoder 接受了远程 URL")
	}
	if img, err := dec(makeDataURI(t, color.NRGBA{R: 255, A: 255})); err != nil {
		t.Fatalf("data: URI 应正常解码: %v", err)
	} else if img.Bounds().Dx() != 1 {
		t.Fatalf("解码宽度 = %d, want 1", img.Bounds().Dx())
	}
}

// TestLocalFileImageDecoderAllowedRoots 验证白名单：目录内放行、目录外拒绝、
// ../ 穿越拒绝、远程 URL 拒绝、data: URI 始终放行。
func TestLocalFileImageDecoderAllowedRoots(t *testing.T) {
	base := t.TempDir()
	allowed := filepath.Join(base, "assets")
	outside := filepath.Join(base, "secret")
	for _, d := range []string{allowed, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writePNGFile := func(p string) {
		t.Helper()
		var buf bytes.Buffer
		one := image.NewNRGBA(image.Rect(0, 0, 1, 1))
		one.SetNRGBA(0, 0, color.NRGBA{G: 255, A: 255})
		if err := png.Encode(&buf, one); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	inFile := filepath.Join(allowed, "ok.png")
	outFile := filepath.Join(outside, "no.png")
	writePNGFile(inFile)
	writePNGFile(outFile)

	dec := LocalFileImageDecoder(allowed)
	if img, err := dec(inFile); err != nil {
		t.Fatalf("白名单内文件应放行: %v", err)
	} else if img.Bounds().Dx() != 1 {
		t.Fatalf("解码宽度 = %d, want 1", img.Bounds().Dx())
	}
	if _, err := dec(outFile); err == nil {
		t.Fatal("白名单外文件被放行")
	}
	if _, err := dec(filepath.Join(allowed, "..", "secret", "no.png")); err == nil {
		t.Fatal("../ 穿越被放行")
	}
	if _, err := dec("https://example.com/x.png"); err == nil {
		t.Fatal("远程 URL 被放行")
	}
	if _, err := dec(makeDataURI(t, color.NRGBA{B: 255, A: 255})); err != nil {
		t.Fatalf("data: URI 应始终放行: %v", err)
	}
}

// TestLocalFileImageDecoderNoRoots 不传白名单时保留旧行为（允许任意本地路径），
// 供完全可信输入迁移使用。
func TestLocalFileImageDecoderNoRoots(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "pic.png")
	var buf bytes.Buffer
	one := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	one.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	if err := png.Encode(&buf, one); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	dec := LocalFileImageDecoder()
	if _, err := dec(file); err != nil {
		t.Fatalf("不传白名单时应允许任意本地路径: %v", err)
	}
	if _, err := dec("ftp://example.com/x.png"); err == nil {
		t.Fatal("远程 URL 仍应被拒绝")
	}
}

// TestRenderUntrustedSVGDoesNotReadFiles 端到端：即使注入 DefaultImageDecoder，
// 不可信 SVG 里的本地路径 href 也不能读文件（不产生像素）。
func TestRenderUntrustedSVGDoesNotReadFiles(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "secret.png")
	var buf bytes.Buffer
	one := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	one.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	if err := png.Encode(&buf, one); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	svg := []byte(`<svg viewBox="0 0 2 2"><image href="` + file + `" x="0" y="0" width="2" height="2"/></svg>`)
	img, err := RenderWithOptions(context.Background(), svg, 8, 8, RenderOptions{ImageDecoder: DefaultImageDecoder()})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if n := nonTransparent(img); n != 0 {
		t.Fatalf("注入安全解码器后仍读到了本地文件（%d 像素）", n)
	}
}
