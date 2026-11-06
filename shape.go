package gosvg

import (
	"strings"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
)

// buildPath 根据基本形状元素构造 rasterx.Path。
// 返回空 path（无子路径）表示该元素无可见图形。
func buildPath(n *node) rasterx.Path {
	var p rasterx.Path
	switch n.name {
	case "circle":
		cx := parseFloatOr(n.attr("cx"), 0)
		cy := parseFloatOr(n.attr("cy"), 0)
		r := parseFloatOr(n.attr("r"), 0)
		if r > 0 {
			rasterx.AddCircle(cx, cy, r, &p)
		}

	case "ellipse":
		cx := parseFloatOr(n.attr("cx"), 0)
		cy := parseFloatOr(n.attr("cy"), 0)
		rx := parseFloatOr(n.attr("rx"), 0)
		ry := parseFloatOr(n.attr("ry"), 0)
		if rx > 0 && ry > 0 {
			rasterx.AddEllipse(cx, cy, rx, ry, 0, &p)
		}

	case "rect":
		x := parseFloatOr(n.attr("x"), 0)
		y := parseFloatOr(n.attr("y"), 0)
		w := parseFloatOr(n.attr("width"), 0)
		h := parseFloatOr(n.attr("height"), 0)
		rx := parseFloatOr(n.attr("rx"), 0)
		ry := parseFloatOr(n.attr("ry"), rx)
		if w > 0 && h > 0 {
			if rx > 0 || ry > 0 {
				rasterx.AddRoundRect(x, y, x+w, y+h, rx, ry, 0, rasterx.RoundGap, &p)
			} else {
				rasterx.AddRect(x, y, x+w, y+h, 0, &p)
			}
		}

	case "line":
		x1 := parseFloatOr(n.attr("x1"), 0)
		y1 := parseFloatOr(n.attr("y1"), 0)
		x2 := parseFloatOr(n.attr("x2"), 0)
		y2 := parseFloatOr(n.attr("y2"), 0)
		p.Start(fpt(x1, y1))
		p.Line(fpt(x2, y2))
		p.Stop(false)

	case "polygon", "polyline":
		pts := parsePoints(n.attr("points"))
		if len(pts) >= 4 {
			p.Start(fpt(pts[0], pts[1]))
			for i := 2; i+1 < len(pts); i += 2 {
				p.Line(fpt(pts[i], pts[i+1]))
			}
			p.Stop(n.name == "polygon")
		}

	case "path":
		p = parsePathData(n.attr("d"))
	}
	return p
}

// parsePoints 解析 "x1,y1 x2,y2 ..." 形式的点列表。
func parsePoints(s string) []float64 {
	var out []float64
	for _, pair := range strings.Fields(s) {
		for _, num := range strings.Split(pair, ",") {
			if v, ok := parseFloat(num); ok {
				out = append(out, v)
			}
		}
	}
	return out
}
