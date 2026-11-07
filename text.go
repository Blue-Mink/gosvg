package gosvg

import (
	"image"
	"math"
	"strings"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// matrixScale 近似返回当前变换的等比缩放因子（取 x/y 缩放的平均）。
func (ctx *renderContext) matrixScale() float64 {
	// Matrix2D 为 [a b c d e f]，等比缩放下 A≈D、B≈C≈0。
	a, b := ctx.matrix.A, ctx.matrix.B
	c, d := ctx.matrix.C, ctx.matrix.D
	sx := math.Hypot(a, b)
	sy := math.Hypot(c, d)
	return (sx + sy) / 2
}

// renderText 渲染 <text> 元素（仅简单单行文本，无 tspans）。
func (ctx *renderContext) renderText(n *node, opacity float64) {
	text := n.content
	if text == "" {
		return
	}
	// 字体解析：内置 Go Regular 为回退；命中 font-family（含系统字体、通用名、
	// 逗号列表）则换用对应字型，字重/字型按 font-weight/font-style 择优。
	f := loadFont()
	if family := n.attr("font-family"); family != "" {
		bold := weightIsBold(n.attr("font-weight"))
		italic := isItalic(n.attr("font-style"))
		if ff := resolveFont(family, bold, italic); ff != nil {
			f = ff
		}
	}
	if f == nil {
		return
	}

	fontSize := parseLengthOr(n.attr("font-size"), 12)
	if fontSize <= 0 {
		return
	}
	anchor := n.attr("text-anchor")
	baseline := n.attr("dominant-baseline")

	// 样式继承（与形状一致）：继承父级 fill，currentColor 归一化；不查 n.attr 裸取，
	// 否则 <g fill="red"><text>…</text></g> 这类继承会失效。
	style := mergeStyle(ctx.style, n)
	fillStr := style.fill
	if strings.EqualFold(fillStr, "currentColor") {
		fillStr = style.color
	}
	// fill="none" / fill="url(#...)"（gradient 等未支持）时 parseColor 失败，
	// 语义应为"不渲染"而非降级成黑色（与 renderShape 的 fill 处理保持一致）。
	c, ok := parseColor(fillStr)
	if !ok || c.A == 0 {
		return
	}

	x := parseFloatOr(n.attr("x"), 0)
	y := parseFloatOr(n.attr("y"), 0)

	// 变换到像素坐标
	px, py := ctx.matrix.Transform(x, y)
	scale := ctx.matrixScale()
	if scale <= 0 {
		scale = 1
	}

	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    fontSize * scale,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return
	}
	defer face.Close()

	// 测量宽度（fixed.Int26_6 → px）
	widthPx := float64(font.MeasureString(face, text)) / 64

	switch anchor {
	case "middle":
		px -= widthPx / 2
	case "end":
		px -= widthPx
	}

	// dominant-baseline：把 y（指定的对齐点）换算成 font.Drawer 需要的基线位置。
	// font.Metrics 的 Descent 为正值（基线到底部的距离），文字中心在基线上方
	// (Ascent-Descent)/2 处；text-before-edge 的 y 指向文字上缘（基线下方 Ascent）。
	switch baseline {
	case "central", "middle":
		m := face.Metrics()
		shift := (m.Ascent - m.Descent) / 2
		py += float64(shift) / 64
	case "text-before-edge", "hanging":
		m := face.Metrics()
		py += float64(m.Ascent) / 64
	}

	dot := fixed.Point26_6{
		X: fixed.Int26_6(px * 64),
		Y: fixed.Int26_6(py * 64),
	}
	drawer := &font.Drawer{
		Dst:  ctx.img,
		Src:  image.NewUniform(rasterx.ApplyOpacity(c, opacity*style.fillOpacity)),
		Face: face,
		Dot:  dot,
	}
	drawer.DrawString(text)
}
