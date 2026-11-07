package gosvg

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
)

// renderMasked 渲染带 mask 引用的元素：先把内容离屏渲染，再用 mask 的
// 亮度（luminance）裁剪 alpha 后合成到主画布。
func (ctx *renderContext) renderMasked(n *node, maskID string, opacity float64) {
	if ctx.nesting >= maxNestingDepth {
		ctx.doc.addDiagnostic("mask/clip nesting limit exceeded; skipped mask")
		ctx.renderNodeBare(n, opacity)
		return
	}
	maskNode := ctx.doc.defs[maskID]
	if maskNode == nil {
		ctx.doc.addDiagnostic("mask reference not found: " + maskID)
		ctx.renderNodeBare(n, opacity)
		return
	}

	bounds := ctx.img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// 1. 渲染目标内容到离屏图像。
	target := ctx.renderSubimage(n, opacity, w, h)

	// 2. 渲染 mask 的亮度图（mask 是容器，亮度由内容决定）。
	maskImg := ctx.renderMaskImage(maskNode, n, w, h)

	// 3. 按 mask 亮度对 target 的 alpha 做乘法合成。
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			s := target.NRGBAAt(x, y)
			m := maskImg.NRGBAAt(x, y)
			lum := luminance(m)
			a := uint8(uint32(s.A) * uint32(lum) / 255)
			if a == 0 {
				continue
			}
			over := color.NRGBA{R: s.R, G: s.G, B: s.B, A: a}
			dst := ctx.img.NRGBAAt(x, y)
			ctx.img.SetNRGBA(x, y, overNRGBA(dst, over))
		}
	}
}

// renderMaskImage 把 mask 元素渲染成独立的 w×h 亮度图。
// 支持 maskContentUnits（子元素坐标系，默认 userSpaceOnUse）、mask 自身
// transform、maskUnits（x/y/width/height 区域，默认 objectBoundingBox）。
func (ctx *renderContext) renderMaskImage(maskNode, targetNode *node, w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))

	contentUnits := strings.ToLower(maskNode.attr("maskcontentunits"))
	maskUnits := strings.ToLower(maskNode.attr("maskunits"))

	// 是否需要被遮罩元素的包围盒（objectBoundingBox 语义）。
	hasRegion := maskNode.hasAttr("x") || maskNode.hasAttr("y") ||
		maskNode.hasAttr("width") || maskNode.hasAttr("height")
	needBB := contentUnits == "objectboundingbox" ||
		(hasRegion && maskUnits != "userspaceonuse")

	var bbX, bbY, bbW, bbH float64
	haveBB := false
	if needBB {
		bbX, bbY, bbW, bbH, haveBB = ctx.elementBounds(targetNode, ctx.matrix, 0)
	}

	// mask 自身 transform 应用在用户空间上。
	maskMatrix := ctx.matrix
	if tr := maskNode.attr("transform"); tr != "" {
		maskMatrix = maskMatrix.Mult(parseTransform(tr))
	}

	// maskContentUnits：objectBoundingBox 时把归一化子元素坐标映射到包围盒。
	contentMatrix := maskMatrix
	if contentUnits == "objectboundingbox" && haveBB {
		contentMatrix = maskMatrix.Mult(rasterx.Identity.Translate(bbX, bbY).Scale(bbW, bbH))
	}

	sub := &renderContext{
		img:     img,
		doc:     ctx.doc,
		matrix:  contentMatrix,
		style:   defaultShapeStyle(),
		ctx:     ctx.ctx,
		opts:    ctx.opts,
		nesting: ctx.nesting + 1,
		err:     ctx.err,
	}
	sub.dasher = newDasherFor(img)
	for _, c := range maskNode.children {
		if err := sub.renderNodeBare(c, 1.0); err != nil && ctx.err == nil {
			ctx.err = err
			break
		}
	}
	if sub.err != nil && ctx.err == nil {
		ctx.err = sub.err
	}

	// mask 区域裁剪（maskUnits）：显式声明了 x/y/width/height 时，区域外亮度归零。
	if hasRegion {
		applyMaskRegion(img, maskNode, maskUnits, maskMatrix, bbX, bbY, bbW, bbH, haveBB)
	}

	return img
}

// applyMaskRegion 把 mask 亮度图裁剪到 [x, y, x+width, y+height] 区域，
// 区域外 alpha 置 0（完全透明）。maskUnits 决定 x/y/width/height 的坐标系。
func applyMaskRegion(img *image.NRGBA, maskNode *node, maskUnits string, maskMatrix rasterx.Matrix2D, bbX, bbY, bbW, bbH float64, haveBB bool) {
	var rx, ry, rw, rh float64
	if maskUnits == "userspaceonuse" {
		// 用户空间坐标，经 maskMatrix 转到 device。
		x := parseLengthOr(maskNode.attr("x"), 0)
		y := parseLengthOr(maskNode.attr("y"), 0)
		wv := parseLengthOr(maskNode.attr("width"), 0)
		hv := parseLengthOr(maskNode.attr("height"), 0)
		px, py := maskMatrix.Transform(x, y)
		qx, qy := maskMatrix.Transform(x+wv, y+hv)
		rx, ry = math.Min(px, qx), math.Min(py, qy)
		rw, rh = math.Abs(qx-px), math.Abs(qy-py)
	} else {
		// objectBoundingBox：归一化值映射到 device 包围盒。
		x := parseFloatOr(maskNode.attr("x"), 0)
		y := parseFloatOr(maskNode.attr("y"), 0)
		wv := parseFloatOr(maskNode.attr("width"), 1)
		hv := parseFloatOr(maskNode.attr("height"), 1)
		if haveBB {
			rx = bbX + x*bbW
			ry = bbY + y*bbH
			rw = wv * bbW
			rh = hv * bbH
		}
	}
	b := img.Bounds()
	for yy := b.Min.Y; yy < b.Max.Y; yy++ {
		for xx := b.Min.X; xx < b.Max.X; xx++ {
			fx, fy := float64(xx), float64(yy)
			if fx < rx || fx > rx+rw || fy < ry || fy > ry+rh {
				img.SetNRGBA(xx, yy, color.NRGBA{})
			}
		}
	}
}

// elementBounds 计算元素 n（含子树）渲染后的 device 空间几何包围盒
// （objectBoundingBox 语义）。忽略 defs/mask/gradient/text 的几何。
// depth 防 use 循环引用。
func (ctx *renderContext) elementBounds(n *node, m rasterx.Matrix2D, depth int) (x, y, w, h float64, ok bool) {
	if depth > maxUseDepth {
		return 0, 0, 0, 0, false
	}
	mm := m
	if tr := n.attr("transform"); tr != "" {
		mm = mm.Mult(parseTransform(tr))
	}

	merge := func(px, py, pw, ph float64, pok bool) {
		if !pok {
			return
		}
		if !ok {
			x, y, w, h, ok = px, py, pw, ph, true
			return
		}
		nx := math.Min(x, px)
		ny := math.Min(y, py)
		mx := math.Max(x+w, px+pw)
		my := math.Max(y+h, py+ph)
		x, y, w, h = nx, ny, mx-nx, my-ny
	}

	switch n.name {
	case "circle", "ellipse", "rect", "line", "path", "polygon", "polyline":
		if p := buildPath(n); len(p) > 0 {
			if px, py, pw, ph, pok := pathBounds(p, mm); pok {
				merge(px, py, pw, ph, pok)
			}
		}
	case "svg", "g":
		for _, c := range n.children {
			if cx, cy, cw, ch, cok := ctx.elementBounds(c, mm, depth+1); cok {
				merge(cx, cy, cw, ch, cok)
			}
		}
	case "use":
		ref := n.attr("href")
		if ref == "" {
			ref = n.attr("xlink:href")
		}
		if len(ref) > 1 && ref[0] == '#' {
			if tgt, ok2 := ctx.doc.defs[ref[1:]]; ok2 {
				mm2 := mm
				if xv, okv := parseFloat(n.attr("x")); okv {
					yv, _ := parseFloat(n.attr("y"))
					mm2 = mm2.Mult(rasterx.Identity.Translate(xv, yv))
				}
				if tx, ty, tw, th, tok := ctx.elementBounds(tgt, mm2, depth+1); tok {
					merge(tx, ty, tw, th, tok)
				}
			}
		}
	}
	return x, y, w, h, ok
}

// renderSubimage 把节点渲染到一个独立的 w×h 透明图像（离屏），
// 复用当前 context 的变换与透明度。
func (ctx *renderContext) renderSubimage(n *node, opacity float64, w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	sub := &renderContext{
		img:     img,
		doc:     ctx.doc,
		matrix:  ctx.matrix,
		style:   ctx.style,
		ctx:     ctx.ctx,
		opts:    ctx.opts,
		nesting: ctx.nesting + 1,
		err:     ctx.err,
	}
	sub.dasher = newDasherFor(img)
	if err := sub.renderNodeBare(n, opacity); err != nil && ctx.err == nil {
		ctx.err = err
	}
	if sub.err != nil && ctx.err == nil {
		ctx.err = sub.err
	}
	return img
}

// luminance 返回 sRGB 像素的感知亮度（0..255），用于 SVG luminance mask。
// 亮度按 alpha 加权：透明像素（alpha=0）即使 RGB 为白，亮度也为 0。
func luminance(c color.NRGBA) uint8 {
	if c.A == 0 {
		return 0
	}
	l := (uint32(c.R)*2126 + uint32(c.G)*7152 + uint32(c.B)*722) / 10000
	return uint8(l * uint32(c.A) / 255)
}

// overNRGBA 按标准 source-over 合成前景到背景。
func overNRGBA(dst, src color.NRGBA) color.NRGBA {
	sa := uint32(src.A)
	da := uint32(dst.A)
	if sa == 0 {
		return dst
	}
	if sa == 255 {
		return src
	}
	outA := sa + da*(255-sa)/255
	if outA == 0 {
		return color.NRGBA{}
	}
	r := (uint32(src.R)*sa*255 + uint32(dst.R)*da*(255-sa)) / (outA * 255)
	g := (uint32(src.G)*sa*255 + uint32(dst.G)*da*(255-sa)) / (outA * 255)
	b := (uint32(src.B)*sa*255 + uint32(dst.B)*da*(255-sa)) / (outA * 255)
	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(outA)}
}

// maskRef 解析 mask="url(#id)" 形式的引用，返回 id（无则空串）。
func maskRef(attr string) string {
	const prefix = "url(#"
	if !strings.HasPrefix(attr, prefix) || !strings.HasSuffix(attr, ")") {
		return ""
	}
	return attr[len(prefix) : len(attr)-1]
}
