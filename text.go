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

// renderText 渲染 <text> 元素（支持 tspan 子元素：直接文本 + 嵌套片段）。
func (ctx *renderContext) renderText(n *node, opacity float64) {
	style := mergeStyle(ctx.style, n)

	fontSize := parseLengthOr(n.attr("font-size"), 12)
	fontFamily := n.attr("font-family")
	fontWeight := n.attr("font-weight")
	fontStyle := n.attr("font-style")

	x := parseFloatOr(n.attr("x"), 0)
	y := parseFloatOr(n.attr("y"), 0)
	anchor := n.attr("text-anchor")
	baseline := n.attr("dominant-baseline")

	cursor := x
	ctx.renderTextSpans(n, style, opacity, fontSize, fontFamily, fontWeight, fontStyle, anchor, baseline, &cursor, y)
}

// renderTextSpans 渲染节点 n 的直接文本与 tspan 子元素，沿 x 游标排布。
func (ctx *renderContext) renderTextSpans(n *node, style shapeStyle, opacity float64, fontSize float64, fontFamily, fontWeight, fontStyle, anchor, baseline string, cursor *float64, y float64) {
	// 直接文本。
	if n.content != "" {
		w := ctx.drawTextSpan(n.content, style, opacity, fontSize, fontFamily, fontWeight, fontStyle, anchor, baseline, *cursor, y)
		*cursor += w
	}
	// tspan 子元素（递归，支持嵌套）。
	for _, c := range n.children {
		if c.name != "tspan" {
			continue
		}
		childStyle := mergeStyle(style, c)
		fs := fontSize
		if v := c.attr("font-size"); v != "" {
			fs = parseLengthOr(v, 12)
		}
		ff := fontFamily
		if v := c.attr("font-family"); v != "" {
			ff = v
		}
		fw := fontWeight
		if v := c.attr("font-weight"); v != "" {
			fw = v
		}
		fst := fontStyle
		if v := c.attr("font-style"); v != "" {
			fst = v
		}
		ca := anchor
		if v := c.attr("text-anchor"); v != "" {
			ca = v
		}
		cb := baseline
		if v := c.attr("dominant-baseline"); v != "" {
			cb = v
		}

		// 位置：默认紧跟游标；dx 相对偏移；x 绝对覆盖。
		startX := *cursor
		if v, ok := parseFloat(c.attr("dx")); ok {
			startX = *cursor + v
		}
		if v, ok := parseFloat(c.attr("x")); ok {
			startX = v
		}
		cy := y
		if v, ok := parseFloat(c.attr("dy")); ok {
			cy = y + v
		}
		if v, ok := parseFloat(c.attr("y")); ok {
			cy = v
		}

		childCursor := startX
		ctx.renderTextSpans(c, childStyle, opacity, fs, ff, fw, fst, ca, cb, &childCursor, cy)
		*cursor = childCursor
	}
}

// drawTextSpan 渲染单个文本片段，返回片段宽度（用户空间单位）。
func (ctx *renderContext) drawTextSpan(text string, style shapeStyle, opacity float64, fontSize float64, fontFamily, fontWeight, fontStyle, anchor, baseline string, x, y float64) float64 {
	if text == "" || fontSize <= 0 {
		return 0
	}
	// 字体解析：内置 Go Regular 为回退；命中 font-family（含系统字体、通用名、
	// 逗号列表）则换用对应字型，字重/字型按 font-weight/font-style 择优。
	f := resolveFace(fontFamily, fontWeight, fontStyle)
	if f == nil {
		return 0
	}

	// fill="none" / fill="url(#...)"（gradient 等未支持）时 parseColor 失败，
	// 语义应为"不渲染"而非降级成黑色（与 renderShape 的 fill 处理保持一致）。
	fillStr := style.fill
	if strings.EqualFold(fillStr, "currentColor") {
		fillStr = style.color
	}
	c, ok := parseColor(fillStr)
	if !ok || c.A == 0 {
		return 0
	}

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
		return 0
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

	return widthPx / scale
}

// MeasureText 测量文本在指定字号下的宽度（用户单位，即与 font-size 同单位）。
// fontFamily 为空时用内置字体；需要字重/字型择优时用 MeasureTextEx。
func MeasureText(text string, fontSize float64, fontFamily string) float64 {
	return measureTextEx(text, fontSize, fontFamily, "", "")
}

// MeasureTextEx 测量文本宽度，可指定 font-family/font-weight/font-style。
func MeasureTextEx(text string, fontSize float64, fontFamily, fontWeight, fontStyle string) float64 {
	return measureTextEx(text, fontSize, fontFamily, fontWeight, fontStyle)
}

func measureTextEx(text string, fontSize float64, fontFamily, fontWeight, fontStyle string) float64 {
	if text == "" || fontSize <= 0 {
		return 0
	}
	f := resolveFace(fontFamily, fontWeight, fontStyle)
	if f == nil {
		return 0
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return 0
	}
	defer face.Close()
	return float64(font.MeasureString(face, text)) / 64
}
