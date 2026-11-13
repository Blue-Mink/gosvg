package gosvg

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
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

// textSpanGeom 是单个 text 片段的设备几何与光栅化器绑定。
// 由 drawTextSpan 构造，供描边路径使用（填充路径仍走旧 font.Drawer）。
type textSpanGeom struct {
	f         *sfnt.Font    // 底层 OpenType 字体（用于 LoadGlyph 取轮廓）
	s         fixed.Int26_6 // face scale（与 opentype.NewFace 同公式）
	px, py    float64       // 设备空间 dot 起点（已含 anchor/baseline 偏移）
	text      string
	widthUser float64 // 宽度换算回用户单位
}

// drawTextSpan 渲染单个文本片段，返回片段宽度（用户空间单位）。
//
// 填充路径保持旧实现（font.Drawer + 位图 mask，零字节变化）。
// 描边路径：字形轮廓（sfnt.LoadGlyph → rasterx.Path）→ strokeCapsules，
// 复用 shape 的精确轮廓描边管线。SVG 语义：fill 在 stroke 之上，
// 因此描边先画、填充后画。
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

	// 描边参数：currentColor 归一化为 color 属性（与 renderShape 一致）。
	strokeStr := style.stroke
	if strings.EqualFold(strokeStr, "currentColor") {
		strokeStr = style.color
	}
	strokeC, strokeOK := parseColor(strokeStr)
	wantStroke := strokeOK && strokeC.A > 0 && style.strokeWidth > 0

	// fill="none" / fill="url(#...)"（gradient 等未支持）时 parseColor 失败，
	// 语义应为"不渲染"而非降级成黑色（与 renderShape 的 fill 处理保持一致）。
	fillStr := style.fill
	if strings.EqualFold(fillStr, "currentColor") {
		fillStr = style.color
	}
	fillC, fillOK := parseColor(fillStr)
	wantFill := fillOK && fillC.A > 0

	// 设备坐标与字号。
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

	widthPx := float64(font.MeasureString(face, text)) / 64
	switch anchor {
	case "middle":
		px -= widthPx / 2
	case "end":
		px -= widthPx
	}
	switch baseline {
	case "central", "middle":
		m := face.Metrics()
		shift := (m.Ascent - m.Descent) / 2
		py += float64(shift) / 64
	case "text-before-edge", "hanging":
		m := face.Metrics()
		py += float64(m.Ascent) / 64
	}

	if wantStroke {
		// 描边先画（fill 在 stroke 之上，与 cairo 一致）。
		geom := &textSpanGeom{
			f:         f,
			s:         faceScale(fontSize * scale),
			px:        px,
			py:        py,
			text:      text,
			widthUser: widthPx / scale,
		}
		ctx.strokeTextSpan(geom, strokeC, style.strokeWidth, style.linecap, style.linejoin, style.miterLimit, style.strokeDasharray, style.strokeDashoffset, opacity*style.strokeOpacity)
	}

	if wantFill {
		dot := fixed.Point26_6{
			X: fixed.Int26_6(px * 64),
			Y: fixed.Int26_6(py * 64),
		}
		drawer := &font.Drawer{
			Dst:  ctx.img,
			Src:  image.NewUniform(rasterx.ApplyOpacity(fillC, opacity*style.fillOpacity)),
			Face: face,
			Dot:  dot,
		}
		drawer.DrawString(text)
	}

	return widthPx / scale
}

// faceScale 与 opentype.NewFace 的 scale 公式一致（0.5 + Size*DPI*64/72）。
func faceScale(size float64) fixed.Int26_6 {
	return fixed.Int26_6(0.5 + size*72*64/72)
}

// strokeTextSpan 把 geom.text 的字形轮廓描边到目标图（设备空间）。
// 每个字形经 sfnt.LoadGlyph 取轮廓（sfnt.Segments）→ rasterx.Path →
// strokeCapsules，复用 shape 描边管线。
//
// 坐标约定：sfnt.Segments 已是像素级（LoadGlyph 内部完成 ppem 缩放与 Y 翻转），
// 因此无需再套 ctx.matrix。字形间推进与 kerning 与 opentype.Face.Glyph 的内部
// 行为逐位一致：当前字形 advance + 前一对的 kern。
func (ctx *renderContext) strokeTextSpan(g *textSpanGeom, strokeC color.NRGBA, strokeWidth float64, linecap, linejoin string, miterLimit float64, dasharray []float64, dashoffset float64, opacity float64) {
	if g == nil || g.f == nil {
		return
	}

	// 逐字形收集 outline。字形起点 px（设备像素）由 px 累加 advance 推进。
	dotX := fixed.Int26_6(g.px * 64)
	dotY := fixed.Int26_6(g.py * 64)

	buf := &sfnt.Buffer{}
	rasterPath := ctx.newTextPath()
	prevIdx := sfnt.GlyphIndex(0)
	prevSet := false

	for _, r := range g.text {
		idx, err := g.f.GlyphIndex(buf, r)
		if err != nil {
			idx = 0 // 未映射字形 → .notdef（与 opentype.Face.Glyph 一致）
		}

		// 字形轮廓：LoadGlyph 返回像素级 Segments（Y 已翻转为向下）。
		// 注意：LoadGlyph 会重置 buf.segments，所以必须**先**加载当前字形、
		// 再推进 dotX，且推进只用当前字形的 advance + 前一对的 kern。
		segs, err := g.f.LoadGlyph(buf, idx, g.s, nil)
		if err == nil && len(segs) > 0 {
			appendGlyphSegments(&rasterPath, segs, dotX, dotY)
		}

		// 推进：advance（当前字形）+ kern（前一对）。
		adv, _ := g.f.GlyphAdvance(buf, idx, g.s, font.HintingFull)
		dotX += adv
		if prevSet {
			if k, kerr := g.f.Kern(buf, prevIdx, idx, g.s, font.HintingFull); kerr == nil {
				dotX += k
			}
		}
		prevIdx, prevSet = idx, true
	}

	if len(rasterPath) == 0 {
		return
	}

	// 描边：全部 cap/join 走精确轮廓（与 strokeColor 同源，tol=0.01/px）。
	// 坐标已是设备空间（sfnt.Segments 像素级 + dot 偏移），传单位矩阵。
	ctx.strokeDevicePath(rasterPath, rasterx.ApplyOpacity(strokeC, opacity), strokeWidth, linecap, linejoin, miterLimit, dasharray, dashoffset)
}

// newTextPath 返回可复用的 rasterx.Path 缓冲（文本描边专用，避免每次分配）。
func (ctx *renderContext) newTextPath() rasterx.Path {
	if ctx.textPath != nil {
		ctx.textPath.Clear()
		return ctx.textPath
	}
	ctx.textPath = make(rasterx.Path, 0, 256)
	return ctx.textPath
}

// appendGlyphSegments 把 sfnt.Segments（像素级、Y 向下）平移到 (ox, oy) 后
// 写入 p。MoveTo 子路径之间用 PathMoveTo 分隔；QuadTo/CubeTo 直接写
// rasterx.Path（rasterx Filler 内部会再 flatten 一次，精度足够）。
func appendGlyphSegments(p *rasterx.Path, segs sfnt.Segments, ox, oy fixed.Int26_6) {
	for _, seg := range segs {
		switch seg.Op {
		case sfnt.SegmentOpMoveTo:
			*p = append(*p, fixed.Int26_6(rasterx.PathMoveTo),
				seg.Args[0].X+ox, seg.Args[0].Y+oy)
		case sfnt.SegmentOpLineTo:
			*p = append(*p, fixed.Int26_6(rasterx.PathLineTo),
				seg.Args[0].X+ox, seg.Args[0].Y+oy)
		case sfnt.SegmentOpQuadTo:
			*p = append(*p, fixed.Int26_6(rasterx.PathQuadTo),
				seg.Args[0].X+ox, seg.Args[0].Y+oy,
				seg.Args[1].X+ox, seg.Args[1].Y+oy)
		case sfnt.SegmentOpCubeTo:
			*p = append(*p, fixed.Int26_6(rasterx.PathCubicTo),
				seg.Args[0].X+ox, seg.Args[0].Y+oy,
				seg.Args[1].X+ox, seg.Args[1].Y+oy,
				seg.Args[2].X+ox, seg.Args[2].Y+oy)
		}
	}
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
