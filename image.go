package gosvg

import (
	"image"
	"strings"

	xdraw "golang.org/x/image/draw"
)

// ImageDecoder 解码 <image> 元素的 href（data: URI、URL、路径等）为位图。
// 由调用方注入；未注入且 RenderOptions.ImageDecoder 也为 nil 时，
// **data: URI 仍会用标准库内置解码**（不涉及文件与网络 IO），
// 其余 href（文件路径 / 远程 URL）静默跳过。
// 这是包级变量，应在渲染前设置一次、不要并发修改；更推荐用
// RenderOptions.ImageDecoder 或 DefaultImageDecoder() 做逐渲染注入。
var ImageDecoder func(href string) (image.Image, error)

// renderImage 渲染 <image> 元素：解码位图并缩放到 x/y/width/height 矩形。
func (ctx *renderContext) renderImage(n *node) {
	href := n.attr("href")
	if href == "" {
		href = n.attr("xlink:href")
	}
	if href == "" {
		ctx.doc.addDiagnostic("<image> skipped: empty href")
		return
	}
	dec := ctx.opts.ImageDecoder
	if dec == nil {
		dec = ImageDecoder
	}
	if dec == nil && strings.HasPrefix(href, "data:") {
		// 内联 data URI 不涉及文件与网络 IO，用标准库直接解码（保持零外部依赖）。
		// 文件路径与远程 URL 仍需调用方显式注入解码器。
		dec = decodeDataURIOnly
	}
	if dec == nil {
		ctx.doc.addDiagnostic("<image> skipped: ImageDecoder not set")
		return
	}
	src, err := dec(href)
	if err != nil || src == nil {
		ctx.doc.addDiagnostic("<image> skipped: decoder failed")
		return
	}

	x := parseLengthOr(n.attr("x"), 0)
	y := parseLengthOr(n.attr("y"), 0)
	width := parseLengthOr(n.attr("width"), 0)
	height := parseLengthOr(n.attr("height"), 0)
	if width <= 0 {
		width = float64(src.Bounds().Dx())
	}
	if height <= 0 {
		height = float64(src.Bounds().Dy())
	}

	// 变换到像素。
	px, py := ctx.matrix.Transform(x, y)
	scale := ctx.matrixScale()
	if scale <= 0 {
		scale = 1
	}
	pw := int(width * scale)
	ph := int(height * scale)
	if pw <= 0 || ph <= 0 {
		return
	}

	dst := ctx.img
	// 裁剪到图像边界。
	xi, yi := int(px), int(py)
	drawRect := image.Rect(xi, yi, xi+pw, yi+ph).Intersect(dst.Bounds())
	if drawRect.Empty() {
		return
	}
	xdraw.CatmullRom.Scale(dst, drawRect, src, src.Bounds(), xdraw.Over, nil)
}
