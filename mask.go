package gosvg

import (
	"image"
	"image/color"
	"strings"
)

// renderMasked 渲染带 mask 引用的元素：先把内容离屏渲染，再用 mask 的
// 亮度（luminance）裁剪 alpha 后合成到主画布。
func (ctx *renderContext) renderMasked(n *node, maskID string, opacity float64) {
	maskNode := ctx.doc.defs[maskID]
	if maskNode == nil {
		// mask 缺失：按无 mask 直接渲染。
		ctx.renderNodeBare(n, opacity)
		return
	}

	bounds := ctx.img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// 1. 渲染目标内容到离屏图像。
	target := ctx.renderSubimage(n, opacity, w, h)

	// 2. 渲染 mask 的子元素（mask 本身是容器，亮度由内容决定）。
	maskImg := ctx.renderMaskImage(maskNode, w, h)

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

// renderMaskImage 把 mask 元素的子元素渲染到独立的 w×h 透明图像，
// 得到亮度图（后续用它裁剪目标 alpha）。
func (ctx *renderContext) renderMaskImage(maskNode *node, w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	sub := &renderContext{
		img:     img,
		doc:     ctx.doc,
		matrix:  ctx.matrix,
		opacity: 1.0,
		style:   defaultShapeStyle(),
	}
	sub.dasher = newDasherFor(img)
	for _, c := range maskNode.children {
		_ = sub.renderNodeBare(c, 1.0)
	}
	return img
}

// renderSubimage 把节点渲染到一个独立的 w×h 透明图像（离屏），
// 复用当前 context 的变换与透明度。
func (ctx *renderContext) renderSubimage(n *node, opacity float64, w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	sub := &renderContext{
		img:     img,
		doc:     ctx.doc,
		matrix:  ctx.matrix,
		opacity: ctx.opacity,
		style:   ctx.style,
	}
	sub.dasher = newDasherFor(img)
	_ = sub.renderNodeBare(n, opacity)
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
