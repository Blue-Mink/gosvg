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
// RenderOptions.ImageDecoder 或 DefaultImageDecoder() 做逐渲染注入（后者仅解
// data: URI；需本地路径请用 LocalFileImageDecoder）。
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

	// preserveAspectRatio：计算 src 等比缩放并对齐后在目标矩形内的实际绘制区
	// fitRect。none 时铺满整个目标（fitRect=full，退化为旧的拉伸逻辑，逐字节不变）；
	// meet 完全放入目标（可能留白）、slice 覆盖目标（可能溢出被裁）。
	sb := src.Bounds()
	sw, sh := float64(sb.Dx()), float64(sb.Dy())
	par := parsePreserveAspectRatio(n.attr("preserveAspectRatio"))
	var fitRect image.Rectangle
	var fx, fy float64 // 目标像素 → src 坐标的映射系数
	// 目标与 src 宽高比在容差内一致时（作者按原图比例设 width/height，语料普遍如此），
	// meet/slice 与拉伸等价。直接退化为 none 可消除 pw/ph 取整带来的 1px 噪声，
	// 保证这类文件（如 C__6ab88 sprite sheet）渲染字节不变。
	aspectMatch := math.Abs(sw/float64(pw)-sh/float64(ph)) < 0.002*sh/float64(ph)
	if par.none || aspectMatch {
		fitRect = full
		fx, fy = sw/float64(pw), sh/float64(ph)
	} else {
		sx, sy := float64(pw)/sw, float64(ph)/sh
		s := math.Min(sx, sy)
		if par.slice {
			s = math.Max(sx, sy)
		}
		fw, fh := sw*s, sh*s
		offX := (float64(pw) - fw) * par.alignX
		offY := (float64(ph) - fh) * par.alignY
		fitRect = image.Rect(
			xi+int(math.Floor(offX)), yi+int(math.Floor(offY)),
			xi+int(math.Floor(offX+fw)), yi+int(math.Floor(offY+fh)),
		)
		fx, fy = sw/fw, sh/fh // meet/slice 等比，二者相等
	}

	// 绘制区 = fitRect ∩ 目标矩形 ∩ 画布。
	// ∩full 保证 slice 溢出目标的部分不绘制；∩fitRect 保证 meet 留白区不绘制。
	drawRect := fitRect.Intersect(full).Intersect(dst.Bounds())
	if drawRect.Empty() {
		return
	}
	// 目标矩形超出画布 / slice 裁剪时，按比例只取 src 的对应切片再缩放到可见区。
	// 若把「已裁剪的 drawRect」直接配完整 src，等于把整张图压缩进可见区（真实语料
	// C__6ab88ea05959193c 是 1696×240 的 sprite sheet，viewBox 只有 240×240，
	// 本应只显示最左侧一个图标，IoU 0.2432）。
	srcRect := image.Rect(
		sb.Min.X+int(math.Floor(float64(drawRect.Min.X-fitRect.Min.X)*fx)),
		sb.Min.Y+int(math.Floor(float64(drawRect.Min.Y-fitRect.Min.Y)*fy)),
		sb.Min.X+int(math.Ceil(float64(drawRect.Max.X-fitRect.Min.X)*fx)),
		sb.Min.Y+int(math.Ceil(float64(drawRect.Max.Y-fitRect.Min.Y)*fy)),
	).Intersect(sb)
	if srcRect.Empty() {
		return
	}
	xdraw.CatmullRom.Scale(dst, drawRect, src, srcRect, xdraw.Over, nil)
}
