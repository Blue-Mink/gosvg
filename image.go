package gosvg

import (
	"image"
	"math"
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
	xi, yi := int(px), int(py)
	full := image.Rect(xi, yi, xi+pw, yi+ph)
	drawRect := full.Intersect(dst.Bounds())
	if drawRect.Empty() {
		return
	}
	// 目标矩形超出画布时，必须按比例只取 src 的对应切片，再把这个切片缩放到
	// 可见矩形。若把「已裁剪的 drawRect」直接配完整 src，等于把整张图压缩进
	// 可见区（真实语料 C__6ab88ea05959193c 是 1696×240 的 sprite sheet，
	// viewBox 只有 240×240，本应只显示最左侧一个图标，IoU 0.2432）。
	sb := src.Bounds()
	fx := float64(sb.Dx()) / float64(pw)
	fy := float64(sb.Dy()) / float64(ph)
	srcRect := image.Rect(
		sb.Min.X+int(math.Floor(float64(drawRect.Min.X-full.Min.X)*fx)),
		sb.Min.Y+int(math.Floor(float64(drawRect.Min.Y-full.Min.Y)*fy)),
		sb.Min.X+int(math.Ceil(float64(drawRect.Max.X-full.Min.X)*fx)),
		sb.Min.Y+int(math.Ceil(float64(drawRect.Max.Y-full.Min.Y)*fy)),
	).Intersect(sb)
	if srcRect.Empty() {
		return
	}
	xdraw.CatmullRom.Scale(dst, drawRect, src, srcRect, xdraw.Over, nil)
}
