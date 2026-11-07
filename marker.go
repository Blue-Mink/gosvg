package gosvg

import (
	"math"
	"strings"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
)

// markerRef 解析 marker-start/mid/end 的 url(#id) 引用，返回 id（无则空串）。
func markerRef(attr string) string {
	const prefix = "url(#"
	if !strings.HasPrefix(attr, prefix) || !strings.HasSuffix(attr, ")") {
		return ""
	}
	return attr[len(prefix) : len(attr)-1]
}

// renderMarkers 在路径的起点/中间顶点/终点放置 marker（箭头等）。
func (ctx *renderContext) renderMarkers(n *node, p rasterx.Path, strokeWidth float64, opacity float64) {
	startRef := markerRef(n.attr("marker-start"))
	midRef := markerRef(n.attr("marker-mid"))
	endRef := markerRef(n.attr("marker-end"))
	if startRef == "" && midRef == "" && endRef == "" {
		return
	}

	scale := ctx.matrixScale()
	if scale <= 0 {
		scale = 1
	}
	// flatten 到像素坐标，直接得到放置点与切线方向。
	subs := flattenPath(p, ctx.matrix, 0.1/scale)

	for _, pts := range subs {
		npts := len(pts)
		if npts < 2 {
			continue
		}
		if startRef != "" {
			a := math.Atan2(pts[1].Y-pts[0].Y, pts[1].X-pts[0].X)
			ctx.renderMarker(startRef, pts[0].X, pts[0].Y, a, strokeWidth, opacity)
		}
		if endRef != "" {
			a := math.Atan2(pts[npts-1].Y-pts[npts-2].Y, pts[npts-1].X-pts[npts-2].X)
			ctx.renderMarker(endRef, pts[npts-1].X, pts[npts-1].Y, a, strokeWidth, opacity)
		}
		if midRef != "" {
			for i := 1; i < npts-1; i++ {
				a := math.Atan2(pts[i].Y-pts[i-1].Y, pts[i].X-pts[i-1].X)
				ctx.renderMarker(midRef, pts[i].X, pts[i].Y, a, strokeWidth, opacity)
			}
		}
	}
}

// renderMarker 渲染单个 marker 到 (px,py)（像素坐标），沿 angle 方向对齐。
func (ctx *renderContext) renderMarker(ref string, px, py, angle, strokeWidth, opacity float64) {
	m := ctx.doc.defs[ref]
	if m == nil || m.name != "marker" {
		return
	}
	units := strings.ToLower(m.attr("markerunits"))
	refX := parseLengthOr(m.attr("refx"), 0)
	refY := parseLengthOr(m.attr("refy"), 0)
	mw := parseLengthOr(m.attr("markerwidth"), 3)
	mh := parseLengthOr(m.attr("markerheight"), 3)
	if mw <= 0 {
		mw = 3
	}
	if mh <= 0 {
		mh = 3
	}

	scale := ctx.matrixScale()
	if scale <= 0 {
		scale = 1
	}
	// viewport 用户单位 → 像素 的缩放因子。
	unit := 1.0
	if units != "userspaceonuse" {
		unit = strokeWidth
		if unit <= 0 {
			unit = 1
		}
	}
	pxScale := unit * scale

	// orient：auto（沿切线）或显式角度（度）。
	if orient := strings.ToLower(m.attr("orient")); orient != "" && orient != "auto" {
		if a, ok := parseFloat(orient); ok {
			angle = a * math.Pi / 180
		}
	}

	// base 把 marker 坐标系（viewBox 或 viewport）变换到像素。
	var base rasterx.Matrix2D = rasterx.Identity
	if vb, ok := parseViewBox(m.attr("viewbox")); ok {
		// viewBox → viewport（mw×mh 用户单位），等比 meet，再缩放至像素。
		sx := mw / vb[2]
		sy := mh / vb[3]
		s := math.Min(sx, sy)
		tx := (mw-vb[2]*s)/2 - vb[0]*s
		ty := (mh-vb[3]*s)/2 - vb[1]*s
		base = rasterx.Identity.Translate(tx*pxScale, ty*pxScale).Scale(s*pxScale, s*pxScale)
	} else {
		// 无 viewBox：marker 坐标系即 viewport（用户单位），直接缩放至像素。
		base = rasterx.Identity.Scale(pxScale, pxScale)
	}

	// refX/refY 对齐点（marker 坐标系）经 base 变换到像素。
	refPx, refPy := base.Transform(refX, refY)

	savedM := ctx.matrix
	savedStyle := ctx.style
	ctx.matrix = rasterx.Identity.Translate(px, py).
		Mult(rasterx.Identity.Rotate(angle)).
		Mult(rasterx.Identity.Translate(-refPx, -refPy)).
		Mult(base)
	ctx.style = mergeStyle(ctx.style, m)
	for _, c := range m.children {
		_ = ctx.renderNode(c, opacity)
	}
	ctx.matrix = savedM
	ctx.style = savedStyle
}
