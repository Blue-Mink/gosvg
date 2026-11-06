package gosvg

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
)

// renderContext 是单次渲染的上下文，持有目标图像、光栅化器和累积的变换/透明度。
type renderContext struct {
	img    *image.NRGBA
	dasher *rasterx.Dasher
	doc    *Document
	matrix rasterx.Matrix2D
	style  shapeStyle // 继承的表现样式（fill/stroke 等）
}

func newRenderContext(width, height int, d *Document) *renderContext {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	scanner := rasterx.NewScannerGV(width, height, img, img.Bounds())
	dasher := rasterx.NewDasher(width, height, scanner)
	ctx := &renderContext{
		img:    img,
		dasher: dasher,
		doc:    d,
		matrix: rasterx.Identity,
		style:  defaultShapeStyle(),
	}
	ctx.matrix = ctx.viewBoxMatrix(width, height)
	return ctx
}

// newDasherFor 为一张离屏图像创建配套的光栅化器。
func newDasherFor(img *image.NRGBA) *rasterx.Dasher {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	return rasterx.NewDasher(w, h, rasterx.NewScannerGV(w, h, img, b))
}

// viewBoxMatrix 计算 viewBox → 目标尺寸的变换（preserveAspectRatio 默认 xMidYMid meet）。
func (ctx *renderContext) viewBoxMatrix(w, h int) rasterx.Matrix2D {
	d := ctx.doc
	if !d.hasVB {
		// 无 viewBox：若声明了 width/height 则按它缩放，否则 1:1。
		vw, vh := d.width, d.height
		if vw <= 0 || vh <= 0 {
			return rasterx.Identity
		}
		return rasterx.Identity.Scale(float64(w)/vw, float64(h)/vh)
	}
	vb := d.viewBox
	sx := float64(w) / vb[2]
	sy := float64(h) / vb[3]
	s := math.Min(sx, sy)
	tx := (float64(w)-vb[2]*s)/2 - vb[0]*s
	ty := (float64(h)-vb[3]*s)/2 - vb[1]*s
	return rasterx.Identity.Translate(tx, ty).Scale(s, s)
}

// renderNode 渲染入口：先处理 mask 引用，再走裸渲染。
func (ctx *renderContext) renderNode(n *node, opacity float64) error {
	// mask 处理：带 mask 引用的元素，离屏渲染 + 亮度合成。
	if id := maskRef(n.attr("mask")); id != "" && n.name != "mask" {
		ctx.renderMasked(n, id, opacity)
		return nil
	}
	return ctx.renderNodeBare(n, opacity)
}

// renderNodeBare 不检查 mask，直接渲染元素及其子树。
// transform / opacity / 样式继承对所有可渲染元素统一生效（含基本形状、text、use），
// 而非仅 svg/g——否则 <rect transform=... opacity=.../> 这类元素级属性会被忽略。
func (ctx *renderContext) renderNodeBare(n *node, opacity float64) error {
	savedM := ctx.matrix
	savedStyle := ctx.style

	// transform 对所有可渲染元素生效。
	if tr := n.attr("transform"); tr != "" {
		ctx.matrix = ctx.matrix.Mult(parseTransform(tr))
	}
	// opacity 对所有可渲染元素生效：本节点有效透明度 = 父级累积 × 自身 opacity。
	nodeOpacity := opacity
	if o := parseFloatOr(n.attr("opacity"), 1); o != 1 {
		nodeOpacity *= o
	}
	ctx.style = mergeStyle(ctx.style, n)

	switch n.name {
	case "svg", "g":
		for _, c := range n.children {
			if err := ctx.renderNode(c, nodeOpacity); err != nil {
				ctx.matrix = savedM
				ctx.style = savedStyle
				return err
			}
		}
	case "defs", "mask", "lineargradient", "radialgradient":
		// 定义类元素不直接渲染。
	case "use":
		ctx.renderUse(n, nodeOpacity)
	case "text":
		ctx.renderText(n, nodeOpacity)
	case "circle", "ellipse", "rect", "line", "path", "polygon", "polyline":
		ctx.renderShape(n, nodeOpacity)
	}

	ctx.matrix = savedM
	ctx.style = savedStyle
	return nil
}

// renderUse 处理 <use href="#id"> 引用。
func (ctx *renderContext) renderUse(n *node, opacity float64) {
	ref := n.attr("href")
	if ref == "" {
		ref = n.attr("xlink:href")
	}
	if len(ref) > 1 && ref[0] == '#' {
		id := ref[1:]
		if target, ok := ctx.doc.defs[id]; ok {
			// 克隆目标，叠加 use 自身的属性（x/y 平移等）。
			savedM := ctx.matrix
			if x, ok := parseFloat(n.attr("x")); ok {
				y, _ := parseFloat(n.attr("y"))
				ctx.matrix = ctx.matrix.Mult(rasterx.Identity.Translate(x, y))
			}
			ctx.renderNode(target, opacity)
			ctx.matrix = savedM
		}
	}
}

// renderShape 渲染一个基本形状元素。
func (ctx *renderContext) renderShape(n *node, opacity float64) {
	p := buildPath(n)
	if len(p) == 0 {
		return
	}

	// 样式解析（继承父级 + 元素自身覆盖）
	style := mergeStyle(ctx.style, n)

	// currentColor 归一化为 color 属性（SVG 默认 black）。
	fillStr, strokeStr := style.fill, style.stroke
	if strings.EqualFold(fillStr, "currentColor") {
		fillStr = style.color
	}
	if strings.EqualFold(strokeStr, "currentColor") {
		strokeStr = style.color
	}

	// 填充
	if c, ok := parseColor(fillStr); ok && c.A > 0 {
		nonZero := style.fillRule != "evenodd"
		ctx.fill(p, c, opacity*style.fillOpacity, nonZero)
	}

	// 描边
	if c, ok := parseColor(strokeStr); ok && c.A > 0 && style.strokeWidth > 0 {
		ctx.stroke(p, c, style.strokeWidth, style.linecap, style.linejoin, opacity*style.strokeOpacity)
	}
}

// shapeStyle 是一次填充/描边所需的有效样式（继承父级后）。
type shapeStyle struct {
	fill          string
	fillRule      string
	fillOpacity   float64
	stroke        string
	strokeWidth   float64
	strokeOpacity float64
	linecap       string
	linejoin      string
	color         string
}

// defaultShapeStyle 返回 SVG 规范默认的表现样式。
func defaultShapeStyle() shapeStyle {
	return shapeStyle{
		fill:          "black",
		fillRule:      "nonzero",
		fillOpacity:   1,
		strokeWidth:   1,
		strokeOpacity: 1,
		linecap:       "butt",
		linejoin:      "miter",
		color:         "black",
	}
}

// mergeStyle 把节点 n 自身的表现属性覆盖到继承样式 base 上。
func mergeStyle(base shapeStyle, n *node) shapeStyle {
	s := base
	if v, ok := n.attrs["fill"]; ok {
		s.fill = v
	}
	if v, ok := n.attrs["fill-rule"]; ok {
		s.fillRule = v
	}
	if v, ok := n.attrs["fill-opacity"]; ok {
		s.fillOpacity = parseFloatOr(v, 1)
	}
	if v, ok := n.attrs["stroke"]; ok {
		s.stroke = v
	}
	if v, ok := n.attrs["stroke-width"]; ok {
		s.strokeWidth = parseFloatOr(v, 1)
	}
	if v, ok := n.attrs["stroke-opacity"]; ok {
		s.strokeOpacity = parseFloatOr(v, 1)
	}
	if v, ok := n.attrs["stroke-linecap"]; ok {
		s.linecap = v
	}
	if v, ok := n.attrs["stroke-linejoin"]; ok {
		s.linejoin = v
	}
	if v, ok := n.attrs["color"]; ok {
		s.color = v
	}
	return s
}

// fill 用指定颜色填充路径。
func (ctx *renderContext) fill(p rasterx.Path, c color.NRGBA, opacity float64, nonZero bool) {
	ctx.dasher.Clear()
	rf := &ctx.dasher.Filler
	rf.SetWinding(nonZero)
	ma := &rasterx.MatrixAdder{Adder: rf, M: ctx.matrix}
	p.AddTo(ma)
	rf.SetColor(rasterx.ApplyOpacity(c, opacity))
	rf.Draw()
	rf.SetWinding(true)
}

// stroke 用指定颜色和宽度描边路径。
func (ctx *renderContext) stroke(p rasterx.Path, c color.NRGBA, width float64, linecap, linejoin string, opacity float64) {
	scale := ctx.matrixScale()
	if scale <= 0 {
		scale = 1
	}
	final := rasterx.ApplyOpacity(c, opacity)

	// 全部 cap/join 组合统一走精确轮廓（单条连续边，解析级 AA）。
	// 用户空间 offset：halfWidth 传用户半宽 width/2，tol 换算到用户单位。
	strokeCapsules(ctx.img, p, ctx.matrix, width/2, final, 0.01/scale, linecap, linejoin)
	return
}
