package gosvg

import (
	"image/color"
	"math"
	"strings"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
)

// gradientRef 解析 fill="url(#id)" 形式的引用，返回 id（无则空串）。
func gradientRef(attr string) string {
	const prefix = "url(#"
	if !strings.HasPrefix(attr, prefix) || !strings.HasSuffix(attr, ")") {
		return ""
	}
	return attr[len(prefix) : len(attr)-1]
}

// hrefRef 解析 href="#id" 形式的引用（gradient 的 href/xlink:href 继承用），
// 返回 id（无则空串）。与 fill="url(#id)" 的 url() 包裹形式不同。
func hrefRef(attr string) string {
	if strings.HasPrefix(attr, "#") && len(attr) > 1 {
		return attr[1:]
	}
	return ""
}

// parseStopOffset 解析 stop 的 offset（0..1 数字或 0%..100% 百分比）。
func parseStopOffset(s string) float64 {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		f, _ := parseFloat(strings.TrimSuffix(s, "%"))
		return f / 100
	}
	return parseFloatOr(s, 0)
}

// resolveGradient 把 fill 值解析为 *rasterx.Gradient（若非 url(#id)、id 不存在、
// 或 id 指向的不是 gradient 元素，均返回 nil）。
//
// 这里**不产生诊断**：同样的 url(#id) 也可能是合法的 <pattern> 引用，是否真的
// 无法解析只能由 render.renderShape 的分派链在「gradient 与 pattern 都未命中」
// 之后判断。此前在此处报 gradient reference not found，会让合法 pattern 被误报。
func (d *Document) resolveGradient(fill string) *rasterx.Gradient {
	id := gradientRef(fill)
	if id == "" {
		return nil
	}
	gn := d.grads[id]
	if gn == nil {
		return nil
	}
	return d.buildGradient(gn, nil)
}

// buildGradient 从 gradient 元素节点构建 rasterx.Gradient。
// seen 用于防止 href 继承循环；href/xlink:href 引用提供默认值（stops 与通用
// 属性），当前元素显式设置的属性覆盖，几何参数始终由当前元素决定。
func (d *Document) buildGradient(n *node, seen map[string]bool) *rasterx.Gradient {
	if seen == nil {
		seen = map[string]bool{}
	}
	var (
		stops  []rasterx.GradStop
		units  = rasterx.ObjectBoundingBox
		spread = rasterx.PadSpread
		matrix = rasterx.Identity
	)

	// href 继承（xlink:href 在解析时已按 Local 合并为 href）。
	if ref := hrefRef(n.attr("href")); ref != "" && !seen[ref] {
		if bn := d.grads[ref]; bn != nil {
			seen[ref] = true
			if base := d.buildGradient(bn, seen); base != nil {
				stops = base.Stops
				units, spread, matrix = base.Units, base.Spread, base.Matrix
			}
		}
	}

	if n.hasAttr("gradientunits") {
		if strings.EqualFold(n.attr("gradientunits"), "userspaceonuse") {
			units = rasterx.UserSpaceOnUse
		} else {
			units = rasterx.ObjectBoundingBox
		}
	}
	if n.hasAttr("spreadmethod") {
		switch strings.ToLower(n.attr("spreadmethod")) {
		case "reflect":
			spread = rasterx.ReflectSpread
		case "repeat":
			spread = rasterx.RepeatSpread
		default:
			spread = rasterx.PadSpread
		}
	}
	if n.hasAttr("gradienttransform") {
		matrix = parseTransform(n.attr("gradienttransform"))
	}

	g := &rasterx.Gradient{
		Units:  units,
		Spread: spread,
		Matrix: matrix,
		Stops:  stops,
	}
	g.IsRadial = n.name == "radialgradient"

	if g.IsRadial {
		cx := parseFloatOr(n.attr("cx"), 0.5)
		cy := parseFloatOr(n.attr("cy"), 0.5)
		r := parseFloatOr(n.attr("r"), 0.5)
		fx := parseFloatOr(n.attr("fx"), cx)
		fy := parseFloatOr(n.attr("fy"), cy)
		g.Points = [5]float64{cx, cy, fx, fy, r}
	} else {
		x1 := parseFloatOr(n.attr("x1"), 0)
		y1 := parseFloatOr(n.attr("y1"), 0)
		x2 := parseFloatOr(n.attr("x2"), 1)
		y2 := parseFloatOr(n.attr("y2"), 0)
		g.Points = [5]float64{x1, y1, x2, y2, 0}
	}

	// 自身有 stop 子元素则覆盖继承的 stops。
	if hasStopChild(n) {
		g.Stops = nil
		for _, c := range n.children {
			if c.name != "stop" {
				continue
			}
			var sc color.NRGBA
			if v, ok := parseColor(c.attr("stop-color")); ok {
				sc = v
			} else {
				sc = color.NRGBA{0, 0, 0, 255}
			}
			g.Stops = append(g.Stops, rasterx.GradStop{
				StopColor: sc,
				Offset:    parseStopOffset(c.attr("offset")),
				Opacity:   parseFloatOr(c.attr("stop-opacity"), 1),
			})
		}
	}

	return g
}

// hasStopChild 判断节点是否有 stop 子元素（决定是否覆盖继承的 stops）。
func hasStopChild(n *node) bool {
	for _, c := range n.children {
		if c.name == "stop" {
			return true
		}
	}
	return false
}

// pathBounds 计算路径经 matrix 变换后的设备空间包围盒（flatten 后取极值）。
func pathBounds(path rasterx.Path, matrix rasterx.Matrix2D) (x, y, w, h float64, ok bool) {
	subs := flattenPath(path, matrix, 0.1)
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, sub := range subs {
		for _, pt := range sub {
			if pt.X < minX {
				minX = pt.X
			}
			if pt.Y < minY {
				minY = pt.Y
			}
			if pt.X > maxX {
				maxX = pt.X
			}
			if pt.Y > maxY {
				maxY = pt.Y
			}
		}
	}
	if minX > maxX {
		return 0, 0, 0, 0, false
	}
	return minX, minY, maxX - minX, maxY - minY, true
}

// fillGradient 用渐变填充路径。
func (ctx *renderContext) fillGradient(p rasterx.Path, g *rasterx.Gradient, opacity float64, nonZero bool) {
	// ObjectBoundingBox 需要设备空间的包围盒。
	if g.Units == rasterx.ObjectBoundingBox {
		x, y, w, h, ok := pathBounds(p, ctx.matrix)
		if !ok {
			return
		}
		g.Bounds.X, g.Bounds.Y, g.Bounds.W, g.Bounds.H = x, y, w, h
	}

	ctx.dasher.Clear()
	rf := &ctx.dasher.Filler
	rf.SetWinding(nonZero)
	ma := &rasterx.MatrixAdder{Adder: rf, M: ctx.matrix}
	p.AddTo(ma)
	rf.SetColor(g.GetColorFunctionUS(opacity, ctx.matrix))
	rf.Draw()
	rf.SetWinding(true)
}
