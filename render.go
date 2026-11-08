package gosvg

import (
	"context"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
)

// renderContext 是单次渲染的上下文，持有目标图像、光栅化器和累积的变换/透明度。
type renderContext struct {
	img         *image.NRGBA
	dasher      *rasterx.Dasher
	doc         *Document
	matrix      rasterx.Matrix2D
	style       shapeStyle // 继承的表现样式（fill/stroke 等）
	useDepth    int        // <use> 嵌套深度（防循环引用栈溢出）
	markerDepth int        // marker 嵌套深度（防 marker 循环引用栈溢出）

	ctx     context.Context // 取消信号（可为 nil）
	opts    RenderOptions   // 本次渲染选项
	nesting int             // 离屏（mask/clip）嵌套层数
	err     error           // 首个取消错误，供所有递归分支快速返回
	visible bool            // CSS visibility 的继承状态（默认 true）

	// strokeFiller 复用描边光栅器：strokeCapsules 此前每次调用都新建
	// ScannerGV+Filler，512px 下每次分配约 1MB float32 缓冲
	// （pprof: setUseFloatingPointMath 占总分配 38.9%）。
	strokeFiller    *rasterx.Filler
	strokeFillerImg *image.NRGBA
}

func newRenderContext(width, height int, d *Document) *renderContext {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	scanner := rasterx.NewScannerGV(width, height, img, img.Bounds())
	dasher := rasterx.NewDasher(width, height, scanner)
	ctx := &renderContext{
		img:     img,
		dasher:  dasher,
		doc:     d,
		matrix:  rasterx.Identity,
		style:   defaultShapeStyle(),
		ctx:     context.Background(),
		opts:    DefaultRenderOptions(),
		visible: true,
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

// viewBoxMatrix 计算 viewBox → 目标尺寸的变换（按 preserveAspectRatio 对齐/缩放）。
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

	// preserveAspectRatio="none"：不保持比例，拉伸填充。
	if d.par.none {
		sx := float64(w) / vb[2]
		sy := float64(h) / vb[3]
		return rasterx.Identity.Translate(-vb[0]*sx, -vb[1]*sy).Scale(sx, sy)
	}

	sx := float64(w) / vb[2]
	sy := float64(h) / vb[3]
	// meet：缩放至完全包含（取较小缩放）；slice：缩放至完全覆盖（取较大缩放，超界裁剪）。
	s := math.Min(sx, sy)
	if d.par.slice {
		s = math.Max(sx, sy)
	}
	tx := (float64(w)-vb[2]*s)*d.par.alignX - vb[0]*s
	ty := (float64(h)-vb[3]*s)*d.par.alignY - vb[1]*s
	return rasterx.Identity.Translate(tx, ty).Scale(s, s)
}

// renderNode 渲染入口：先处理 clip-path，再处理 mask，最后走裸渲染。
func (ctx *renderContext) renderNode(n *node, opacity float64) error {
	if ctx.err != nil {
		return ctx.err
	}
	if ctx.ctx != nil {
		if err := ctx.ctx.Err(); err != nil {
			ctx.err = err
			return err
		}
	}
	// CSS display:none 跳过整棵子树（含子元素），不产生任何绘制。
	if strings.EqualFold(n.attr("display"), "none") {
		return nil
	}
	// clip-path 处理：带 clip 引用的元素，离屏渲染 + 形状 alpha 裁剪。
	if id := clipPathRef(n.attr("clip-path")); id != "" && n.name != "clippath" {
		ctx.renderClipped(n, id, opacity)
		return nil
	}
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
	// display:none 跳过整棵子树（renderNode 已挡一层，这里兜住直接调用方）。
	if strings.EqualFold(n.attr("display"), "none") {
		return nil
	}
	// visibility 是继承属性：hidden 只跳过本节点绘制，子元素可显式 visible 覆盖。
	vis := ctx.visible
	switch strings.ToLower(n.attr("visibility")) {
	case "hidden", "collapse":
		vis = false
	case "visible":
		vis = true
	}
	savedVis := ctx.visible
	ctx.visible = vis
	defer func() { ctx.visible = savedVis }()

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
	case "defs", "mask", "clippath", "lineargradient", "radialgradient", "marker", "pattern", "style":
		// 定义类元素不直接渲染。
	case "use":
		if ctx.visible {
			ctx.renderUse(n, nodeOpacity)
		}
	case "image":
		if ctx.visible {
			ctx.renderImage(n)
		}
	case "text":
		if ctx.visible {
			ctx.renderText(n, nodeOpacity)
		}
	case "circle", "ellipse", "rect", "line", "path", "polygon", "polyline":
		if ctx.visible {
			ctx.renderShape(n, nodeOpacity)
		}
	default:
		// 无害的元数据元素不记诊断（合法 SVG，只是不参与渲染）。
		if !isMetadataElement(n.name) {
			ctx.doc.addDiagnostic("unsupported element skipped: " + n.name)
		}
	}

	ctx.matrix = savedM
	ctx.style = savedStyle
	return nil
}

// maxUseDepth 是 <use> 引用的最大嵌套深度，用于防止循环引用导致栈溢出。
const maxUseDepth = 100

// maxMarkerDepth 是 marker 引用的最大嵌套深度，用于防止 marker 循环引用导致栈溢出。
const maxMarkerDepth = 100

// renderUse 处理 <use href="#id"> 引用。
func (ctx *renderContext) renderUse(n *node, opacity float64) {
	// 循环引用（如 <use id="a" href="#a"/> 或 a↔b 互引）会无限递归导致栈溢出，
	// 用嵌套深度上限截断。正常 SVG 的 use 嵌套远小于该值。
	if ctx.useDepth >= maxUseDepth {
		return
	}
	ctx.useDepth++
	defer func() { ctx.useDepth-- }()

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
		} else {
			ctx.doc.addDiagnostic("use reference not found: " + id)
		}
	} else {
		ctx.doc.addDiagnostic("use skipped: invalid href")
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
	} else if g := ctx.doc.resolveGradient(fillStr); g != nil {
		nonZero := style.fillRule != "evenodd"
		ctx.fillGradient(p, g, opacity*style.fillOpacity, nonZero)
	} else if pat := ctx.doc.resolvePattern(fillStr); pat != nil {
		nonZero := style.fillRule != "evenodd"
		ctx.fillPattern(p, pat, opacity*style.fillOpacity, nonZero)
	}

	// 描边
	if c, ok := parseColor(strokeStr); ok && c.A > 0 && style.strokeWidth > 0 {
		ctx.stroke(p, c, style.strokeWidth, style.linecap, style.linejoin, style.miterLimit, style.strokeDasharray, style.strokeDashoffset, opacity*style.strokeOpacity)
	} else if g := ctx.doc.resolveGradient(strokeStr); g != nil && style.strokeWidth > 0 {
		ctx.strokeGradient(p, g, style.strokeWidth, style.linecap, style.linejoin, style.miterLimit, style.strokeDasharray, style.strokeDashoffset, opacity*style.strokeOpacity)
	} else if pat := ctx.doc.resolvePattern(strokeStr); pat != nil && style.strokeWidth > 0 {
		// stroke="url(#pattern)"：先按纯色黑描边到掩膜再平铺不划算，
		// 这里退化为用 pattern 填充描边路径的包围盒不可行，故按规范少见用法记诊断。
		ctx.doc.addDiagnostic("stroke with pattern reference is not supported")
	}

	// marker（箭头）
	ctx.renderMarkers(n, p, style.strokeWidth, opacity)
}

// shapeStyle 是一次填充/描边所需的有效样式（继承父级后）。
type shapeStyle struct {
	fill             string
	fillRule         string
	fillOpacity      float64
	stroke           string
	strokeWidth      float64
	strokeOpacity    float64
	linecap          string
	linejoin         string
	miterLimit       float64
	strokeDasharray  []float64
	strokeDashoffset float64
	color            string
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
		miterLimit:    4,
		color:         "black",
	}
}

// mergeStyle 把节点 n 自身的表现属性覆盖到继承样式 base 上。
func mergeStyle(base shapeStyle, n *node) shapeStyle {
	s := base
	if v, ok := n.attrs["fill"]; ok && isValidPaint(v) {
		s.fill = v
	}
	if v, ok := n.attrs["fill-rule"]; ok {
		s.fillRule = v
	}
	if v, ok := n.attrs["fill-opacity"]; ok {
		s.fillOpacity = parseFloatOr(v, 1)
	}
	if v, ok := n.attrs["stroke"]; ok && isValidPaint(v) {
		s.stroke = v
	}
	if v, ok := n.attrs["stroke-width"]; ok {
		s.strokeWidth = parseLengthOr(v, 1)
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
	if v, ok := n.attrs["stroke-miterlimit"]; ok {
		s.miterLimit = parseLengthOr(v, 4)
	}
	if v, ok := n.attrs["stroke-dasharray"]; ok {
		if da, ok := parseDashArray(v); ok {
			s.strokeDasharray = da
		} else {
			s.strokeDasharray = nil
		}
	}
	if v, ok := n.attrs["stroke-dashoffset"]; ok {
		s.strokeDashoffset = parseFloatOr(v, 0)
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

// stroke 用指定纯色描边路径。
func (ctx *renderContext) stroke(p rasterx.Path, c color.NRGBA, width float64, linecap, linejoin string, miterLimit float64, dasharray []float64, dashoffset float64, opacity float64) {
	ctx.strokeColor(p, rasterx.ApplyOpacity(c, opacity), width, linecap, linejoin, miterLimit, dasharray, dashoffset)
}

// strokeGradient 用渐变描边路径（stroke="url(#id)"）。
func (ctx *renderContext) strokeGradient(p rasterx.Path, g *rasterx.Gradient, width float64, linecap, linejoin string, miterLimit float64, dasharray []float64, dashoffset float64, opacity float64) {
	// objectBoundingBox 用路径几何包围盒（不含 stroke 宽度）。
	if g.Units == rasterx.ObjectBoundingBox {
		if x, y, w, h, ok := pathBounds(p, ctx.matrix); ok {
			g.Bounds.X, g.Bounds.Y, g.Bounds.W, g.Bounds.H = x, y, w, h
		}
	}
	ctx.strokeColor(p, g.GetColorFunctionUS(opacity, ctx.matrix), width, linecap, linejoin, miterLimit, dasharray, dashoffset)
}

// strokeColor 通用描边：clr 可以是 color.Color 或 ColorFunc（渐变）。
func (ctx *renderContext) strokeColor(p rasterx.Path, clr interface{}, width float64, linecap, linejoin string, miterLimit float64, dasharray []float64, dashoffset float64) {
	scale := ctx.matrixScale()
	if scale <= 0 {
		scale = 1
	}

	// 全部 cap/join 组合统一走精确轮廓（单条连续边，解析级 AA）。
	// 用户空间 offset：halfWidth 传用户半宽 width/2，tol 换算到用户单位。
	strokeCapsules(ctx.strokeFillerFor(ctx.img), p, ctx.matrix, width/2, clr, 0.01/scale, linecap, linejoin, miterLimit, dasharray, dashoffset)
	return
}

// isMetadataElement 返回不参与渲染、但完全合法的 SVG 元数据元素名。
// 这些元素被静默忽略属于正常行为，不应作为「不支持」诊断报给调用方。
func isMetadataElement(name string) bool {
	switch name {
	case "title", "desc", "metadata", "namedview":
		return true
	}
	return false
}

// isValidPaint 判断 fill / stroke 的值是否可识别。
// 按 CSS 语义，**无效声明应被忽略并保持继承值**，而不是当成「不填充」——
// 否则 fill="000000"（缺 #，Material Symbols 等工具的真实导出写法）会让整个
// 图形消失。实测：修复前该写法渲染为全透明，修复后回退到规范默认 black。
func isValidPaint(v string) bool {
	t := strings.TrimSpace(v)
	if t == "" {
		return false
	}
	switch strings.ToLower(t) {
	case "none", "transparent", "currentcolor":
		return true
	}
	if l := strings.ToLower(t); strings.HasPrefix(l, "url(") && strings.Contains(l, ")") {
		return true // url(#id) 以及 SVG 1.1 的 "url(#id) <fallback>" 写法
	}
	_, ok := parseColor(t)
	return ok
}

// strokeFillerFor 返回绑定到 img 的可复用 Filler，避免每次描边都重新分配
// 光栅器缓冲（见 renderContext.strokeFiller 注释）。
func (ctx *renderContext) strokeFillerFor(img *image.NRGBA) *rasterx.Filler {
	if ctx.strokeFiller != nil && ctx.strokeFillerImg == img {
		return ctx.strokeFiller
	}
	b := img.Bounds()
	f := rasterx.NewFiller(b.Dx(), b.Dy(), rasterx.NewScannerGV(b.Dx(), b.Dy(), img, b))
	ctx.strokeFiller = f
	ctx.strokeFillerImg = img
	return f
}
