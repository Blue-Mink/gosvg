package gosvg

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
)

// maxPatternTile 限制单个 pattern tile 的边长（像素），防止
// patternUnits=userSpaceOnUse 配合超大 width/height 造成巨量分配。
const maxPatternTile = 4096

// resolvePattern 把 fill/stroke 值解析为 <pattern> 元素节点；
// 引用的不是 pattern（或不存在）时返回 nil。
func (d *Document) resolvePattern(v string) *node {
	id := gradientRef(v) // 复用 url(#id) 解析
	if id == "" {
		return nil
	}
	n := d.defs[id]
	if n == nil || n.name != "pattern" {
		return nil
	}
	return n
}

// fillPattern 用 <pattern> 的平铺内容填充路径。
//
// 支持：patternUnits（userSpaceOnUse / objectBoundingBox，默认后者）、
// patternContentUnits（默认 userSpaceOnUse）、patternTransform、tile 内的 viewBox
// （等比拉伸，不做 preserveAspectRatio 对齐）、tile 内的 <use> / <image> 等元素。
func (ctx *renderContext) fillPattern(p rasterx.Path, pat *node, opacity float64, nonZero bool) {
	if opacity <= 0 {
		return
	}
	bx, by, bw, bh, ok := pathBounds(p, ctx.matrix)
	if !ok || bw <= 0 || bh <= 0 {
		return
	}
	scale := ctx.matrixScale()
	if scale <= 0 {
		scale = 1
	}

	pw := parseLengthOr(pat.attr("width"), 0)
	ph := parseLengthOr(pat.attr("height"), 0)
	if pw <= 0 || ph <= 0 {
		ctx.doc.addDiagnostic("pattern skipped: non-positive tile size")
		return
	}

	// tile 在 device 空间的尺寸与平铺原点
	var tw, th, ox, oy float64
	if strings.EqualFold(pat.attr("patternunits"), "userspaceonuse") {
		tw, th = pw*scale, ph*scale
		ux, uy := parseLengthOr(pat.attr("x"), 0), parseLengthOr(pat.attr("y"), 0)
		ox, oy = ctx.matrix.Transform(ux, uy)
	} else { // objectBoundingBox（SVG 默认）
		tw, th = pw*bw, ph*bh
		ox = bx + parseLengthOr(pat.attr("x"), 0)*bw
		oy = by + parseLengthOr(pat.attr("y"), 0)*bh
	}
	tileW, tileH := int(math.Ceil(tw)), int(math.Ceil(th))
	if tileW <= 0 || tileH <= 0 || tileW > maxPatternTile || tileH > maxPatternTile {
		ctx.doc.addDiagnostic("pattern skipped: tile size out of range")
		return
	}

	tile := ctx.renderPatternTile(pat, tileW, tileH, bw, bh, scale)
	if tile == nil {
		return
	}
	mask := ctx.renderPathMask(p, bx, by, bw, bh, opacity, nonZero)
	if mask == nil {
		return
	}

	r := image.Rect(int(math.Floor(bx)), int(math.Floor(by)),
		int(math.Ceil(bx+bw)), int(math.Ceil(by+bh))).Intersect(ctx.img.Bounds())
	if r.Empty() {
		return
	}
	// 按 tile 位置逐块 DrawMask。不能用「平铺 image + 一次 DrawMask」：
	// draw.DrawMask 会用 src.Bounds() 裁剪目标矩形，而平铺 image 的 Bounds 只有
	// 一个 tile 大，结果只画出左上角一块（实测 64/256 而非全覆盖）。
	oxi, oyi := int(math.Round(ox)), int(math.Round(oy))
	startI := int(math.Floor(float64(r.Min.X-oxi) / float64(tileW)))
	startJ := int(math.Floor(float64(r.Min.Y-oyi) / float64(tileH)))
	for j := startJ; ; j++ {
		ty := oyi + j*tileH
		if ty >= r.Max.Y {
			break
		}
		for i := startI; ; i++ {
			tx := oxi + i*tileW
			if tx >= r.Max.X {
				break
			}
			tr := image.Rect(tx, ty, tx+tileW, ty+tileH).Intersect(r)
			if tr.Empty() {
				continue
			}
			sp := image.Point{X: tr.Min.X - tx, Y: tr.Min.Y - ty}
			mp := image.Point{X: tr.Min.X - r.Min.X, Y: tr.Min.Y - r.Min.Y}
			draw.DrawMask(ctx.img, tr, tile, sp, mask, mp, draw.Over)
		}
	}
}

// renderPatternTile 把 pattern 的子元素渲染成一块 tileW×tileH 的位图。
func (ctx *renderContext) renderPatternTile(pat *node, tileW, tileH int, bw, bh, scale float64) *image.NRGBA {
	tile := image.NewNRGBA(image.Rect(0, 0, tileW, tileH))

	var m rasterx.Matrix2D
	switch {
	case pat.attr("viewbox") != "":
		if v, ok := parseViewBox(pat.attr("viewbox")); ok && v[2] > 0 && v[3] > 0 {
			m = rasterx.Identity.Translate(-v[0], -v[1]).
				Scale(float64(tileW)/v[2], float64(tileH)/v[3])
		} else {
			m = rasterx.Identity.Scale(scale, scale)
		}
	case strings.EqualFold(pat.attr("patterncontentunits"), "objectboundingbox"):
		m = rasterx.Identity.Scale(bw, bh)
	default:
		m = rasterx.Identity.Scale(scale, scale)
	}
	if pt := pat.attr("patterntransform"); pt != "" {
		m = m.Mult(parseTransform(pt))
	}

	sub := &renderContext{
		img:     tile,
		doc:     ctx.doc,
		matrix:  m,
		style:   defaultShapeStyle(),
		ctx:     ctx.ctx,
		opts:    ctx.opts,
		nesting: ctx.nesting + 1,
		err:     ctx.err,
		visible: ctx.visible,
	}
	sub.dasher = newDasherFor(tile)
	for _, c := range pat.children {
		if err := sub.renderNodeBare(c, 1.0); err != nil {
			if ctx.err == nil {
				ctx.err = err
			}
			return nil
		}
	}
	if sub.err != nil && ctx.err == nil {
		ctx.err = sub.err
	}
	return tile
}

// renderPathMask 把路径填充成 bbox 局部坐标的 alpha 掩膜，opacity 乘进 alpha。
func (ctx *renderContext) renderPathMask(p rasterx.Path, bx, by, bw, bh, opacity float64, nonZero bool) *image.NRGBA {
	w, h := int(math.Ceil(bw))+1, int(math.Ceil(bh))+1
	if w <= 0 || h <= 0 {
		return nil
	}
	mask := image.NewNRGBA(image.Rect(0, 0, w, h))
	sub := &renderContext{
		img:    mask,
		doc:    ctx.doc,
		matrix: rasterx.Identity.Translate(-bx, -by).Mult(ctx.matrix),
		style:  defaultShapeStyle(),
		ctx:    ctx.ctx,
		opts:   ctx.opts,
		// 掩膜只关心路径覆盖，不受父级 visibility 影响
		visible: true,
		err:     ctx.err,
	}
	sub.dasher = newDasherFor(mask)
	sub.fill(p, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, opacity, nonZero)
	return mask
}
