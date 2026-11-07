package gosvg

import (
	"image"
	"image/color"
	"strings"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
)

// clipPathRef 解析 clip-path="url(#id)" 形式的引用，返回 id（无则空串）。
func clipPathRef(attr string) string {
	const prefix = "url(#"
	if !strings.HasPrefix(attr, prefix) || !strings.HasSuffix(attr, ")") {
		return ""
	}
	return attr[len(prefix) : len(attr)-1]
}

// renderClipped 渲染带 clip-path 引用的元素：离屏渲染目标内容，用 clipPath
// 形状的 alpha 裁剪后合成。与 mask 不同，clip 只用形状覆盖（alpha），不做
// 亮度换算。
func (ctx *renderContext) renderClipped(n *node, clipID string, opacity float64) {
	if ctx.nesting >= maxNestingDepth {
		ctx.doc.addDiagnostic("mask/clip nesting limit exceeded; skipped clip")
		ctx.renderNodeBare(n, opacity)
		return
	}
	clipNode := ctx.doc.defs[clipID]
	if clipNode == nil || clipNode.name != "clippath" {
		ctx.doc.addDiagnostic("clip-path reference not found: " + clipID)
		ctx.renderNodeBare(n, opacity)
		return
	}

	bounds := ctx.img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	target := ctx.renderSubimage(n, opacity, w, h)
	clipImg := ctx.renderClipMask(clipNode, n, w, h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			s := target.NRGBAAt(x, y)
			c := clipImg.NRGBAAt(x, y)
			a := uint8(uint32(s.A) * uint32(c.A) / 255)
			if a == 0 {
				continue
			}
			over := color.NRGBA{R: s.R, G: s.G, B: s.B, A: a}
			dst := ctx.img.NRGBAAt(x, y)
			ctx.img.SetNRGBA(x, y, overNRGBA(dst, over))
		}
	}
}

// renderClipMask 把 clipPath 元素渲染成独立的 w×h alpha 图：子元素形状覆盖
// 处 alpha=255，形状外 alpha=0。clipPathUnits 默认 userSpaceOnUse（不同于
// mask 的 maskUnits 默认 objectBoundingBox）。
func (ctx *renderContext) renderClipMask(clipNode, targetNode *node, w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))

	units := strings.ToLower(clipNode.attr("clippathunits"))

	hasRegion := clipNode.hasAttr("x") || clipNode.hasAttr("y") ||
		clipNode.hasAttr("width") || clipNode.hasAttr("height")
	needBB := units == "objectboundingbox" || (hasRegion && units != "userspaceonuse")

	var bbX, bbY, bbW, bbH float64
	haveBB := false
	if needBB {
		bbX, bbY, bbW, bbH, haveBB = ctx.elementBounds(targetNode, ctx.matrix, 0)
	}

	// clipPath 自身 transform 应用在用户空间上。
	maskMatrix := ctx.matrix
	if tr := clipNode.attr("transform"); tr != "" {
		maskMatrix = maskMatrix.Mult(parseTransform(tr))
	}
	// clipPathUnits：objectBoundingBox 时把归一化子元素坐标映射到包围盒。
	contentMatrix := maskMatrix
	if units == "objectboundingbox" && haveBB {
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
		visible: ctx.visible,
	}
	sub.dasher = newDasherFor(img)
	for _, c := range clipNode.children {
		if err := sub.renderNodeBare(c, 1.0); err != nil && ctx.err == nil {
			ctx.err = err
			break
		}
	}
	if sub.err != nil && ctx.err == nil {
		ctx.err = sub.err
	}

	if hasRegion {
		applyMaskRegion(img, clipNode, units, maskMatrix, bbX, bbY, bbW, bbH, haveBB)
	}

	return img
}
