package gosvg

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
)

// EncodeImage 把位图按 format 编码写入 w。
// format 支持 "png"（默认）与 "jpeg"/"jpg"；JPEG 无 alpha 通道，会先把图像
// 合成到 bg 上（bg 为 nil 时用白色），避免出现黑底。
func EncodeImage(w io.Writer, img *image.NRGBA, format string, quality int, bg color.Color) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "png":
		return png.Encode(w, img)
	case "jpeg", "jpg":
		return jpeg.Encode(w, flattenForJPEG(img, bg), jpegOptions(quality))
	default:
		return fmt.Errorf("gosvg: unsupported output format %q (want png or jpeg)", format)
	}
}

// RenderToWriter 渲染 SVG 并把结果按 format 编码写入 w（一步到位）。
func RenderToWriter(w io.Writer, data []byte, width, height int, format string, opts RenderOptions) error {
	img, err := RenderWithOptions(opts.contextOrBackground(), data, width, height, opts)
	if err != nil {
		return err
	}
	return EncodeImage(w, img, format, opts.JPEGQuality, opts.Background)
}

func jpegOptions(quality int) *jpeg.Options {
	if quality <= 0 {
		quality = 90
	}
	if quality > 100 {
		quality = 100
	}
	return &jpeg.Options{Quality: quality}
}

// flattenForJPEG 把带 alpha 的 NRGBA 合成到不透明背景上，供 JPEG 编码。
func flattenForJPEG(img *image.NRGBA, bg color.Color) *image.RGBA {
	if bg == nil {
		bg = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}
	br, bgg, bb, ba := bg.RGBA()
	if ba == 0 {
		br, bgg, bb = 0xffff, 0xffff, 0xffff // 全透明背景按白色处理
	}
	out := image.NewRGBA(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := img.NRGBAAt(x, y)
			a := float64(c.A) / 255
			ia := 1 - a
			out.SetRGBA(x, y, color.RGBA{
				R: uint8((float64(c.R)*a + float64(br>>8)*ia) + 0.5),
				G: uint8((float64(c.G)*a + float64(bgg>>8)*ia) + 0.5),
				B: uint8((float64(c.B)*a + float64(bb>>8)*ia) + 0.5),
				A: 255,
			})
		}
	}
	return out
}
