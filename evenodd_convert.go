package gosvg

import (
	"math"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
	"golang.org/x/image/math/fixed"
)

// evenodd_convert.go 把 fill-rule="evenodd" 的多子路径 shape 等价转换成
// 「方向交替的单条 path + nonzero」，绕开光栅器在多子路径 even-odd 上的偏差。
//
// 背景：CAIROSVG_IOU.md 修正 6（PoC）/ 修正 7（Go 侧落地）记录，语料中 11 个
// IoU<0.5 的文件全部
// 同时声明 fill-rule/clip-rule="evenodd"，且都含 2–4 个 Z 子路径；外部 SVG
// 预处理实验证明，仅调整子路径方向并改用 nonzero 即可把平均 IoU 从 0.2533
// 提到 0.9924，全量 61 个改动文件 30 提升 / 31 不变 / 0 退化。
//
// 这里在「解析层」而非光栅器核心实现：输入是 parsePathData 已经归一化成
// 绝对坐标的 rasterx.Path，输出仍是 rasterx.Path，internal/rasterx 与
// internal/vector 一行不改。转换只作用于填充路径，描边与 marker 仍用原 path。

// eoSeg 是一条不含起点命令的段；pts 存控制点与终点（Close 段为空）。
type eoSeg struct {
	cmd rasterx.PathCommand
	pts []fixed.Point26_6
}

type eoSub struct {
	start  fixed.Point26_6
	segs   []eoSeg
	closed bool
}

// eoPt 是参与几何判断的浮点点（避免 fixed.Int26_6 乘法溢出）。
type eoPt struct{ x, y float64 }

func eoFixed(v fixed.Int26_6) float64 { return float64(v) / 64 }

func eoPoint(x, y fixed.Int26_6) fixed.Point26_6 {
	return fixed.Point26_6{X: x, Y: y}
}

// splitEOSubpaths 按 PathMoveTo 把扁平 Path 切成子路径。
func splitEOSubpaths(p rasterx.Path) []eoSub {
	var subs []eoSub
	idx := -1
	for i := 0; i < len(p); {
		switch rasterx.PathCommand(p[i]) {
		case rasterx.PathMoveTo:
			subs = append(subs, eoSub{start: eoPoint(p[i+1], p[i+2])})
			idx = len(subs) - 1
			i += 3
		case rasterx.PathLineTo:
			if idx >= 0 {
				subs[idx].segs = append(subs[idx].segs, eoSeg{
					cmd: rasterx.PathLineTo,
					pts: []fixed.Point26_6{eoPoint(p[i+1], p[i+2])},
				})
			}
			i += 3
		case rasterx.PathQuadTo:
			if idx >= 0 {
				subs[idx].segs = append(subs[idx].segs, eoSeg{
					cmd: rasterx.PathQuadTo,
					pts: []fixed.Point26_6{
						eoPoint(p[i+1], p[i+2]),
						eoPoint(p[i+3], p[i+4]),
					},
				})
			}
			i += 5
		case rasterx.PathCubicTo:
			if idx >= 0 {
				subs[idx].segs = append(subs[idx].segs, eoSeg{
					cmd: rasterx.PathCubicTo,
					pts: []fixed.Point26_6{
						eoPoint(p[i+1], p[i+2]),
						eoPoint(p[i+3], p[i+4]),
						eoPoint(p[i+5], p[i+6]),
					},
				})
			}
			i += 7
		case rasterx.PathClose:
			if idx >= 0 {
				subs[idx].closed = true
				subs[idx].segs = append(subs[idx].segs, eoSeg{cmd: rasterx.PathClose})
			}
			i++
		default:
			// 未知命令：放弃整条 path 的转换，保持原样。
			return nil
		}
	}
	return subs
}

// eoSamplePoly 把子路径展开成折线多边形（曲线按 8 段采样），用于面积和
// 点在多边形内的判断。Close 段不需要额外点：shoelace 会自动闭合首尾。
func eoSamplePoly(s eoSub) []eoPt {
	poly := []eoPt{{x: eoFixed(s.start.X), y: eoFixed(s.start.Y)}}
	cur := s.start
	for _, seg := range s.segs {
		switch seg.cmd {
		case rasterx.PathLineTo:
			cur = seg.pts[0]
			poly = append(poly, eoPt{eoFixed(cur.X), eoFixed(cur.Y)})
		case rasterx.PathQuadTo:
			p0x, p0y := eoFixed(cur.X), eoFixed(cur.Y)
			cx, cy := eoFixed(seg.pts[0].X), eoFixed(seg.pts[0].Y)
			ex, ey := eoFixed(seg.pts[1].X), eoFixed(seg.pts[1].Y)
			for k := 1; k <= 8; k++ {
				t := float64(k) / 8
				u := 1 - t
				poly = append(poly, eoPt{
					u*u*p0x + 2*u*t*cx + t*t*ex,
					u*u*p0y + 2*u*t*cy + t*t*ey,
				})
			}
			cur = seg.pts[1]
		case rasterx.PathCubicTo:
			p0x, p0y := eoFixed(cur.X), eoFixed(cur.Y)
			c1x, c1y := eoFixed(seg.pts[0].X), eoFixed(seg.pts[0].Y)
			c2x, c2y := eoFixed(seg.pts[1].X), eoFixed(seg.pts[1].Y)
			ex, ey := eoFixed(seg.pts[2].X), eoFixed(seg.pts[2].Y)
			for k := 1; k <= 8; k++ {
				t := float64(k) / 8
				u := 1 - t
				a, b, c, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
				poly = append(poly, eoPt{
					a*p0x + b*c1x + c*c2x + d*ex,
					a*p0y + b*c1y + c*c2y + d*ey,
				})
			}
			cur = seg.pts[2]
		case rasterx.PathClose:
			cur = s.start
		}
	}
	return poly
}

// eoSignedArea 返回 shoelace 有向面积（y 轴向下的屏幕坐标下，符号只用于
// 相对比较，不解释成几何「顺/逆时针」）。
func eoSignedArea(poly []eoPt) float64 {
	if len(poly) < 3 {
		return 0
	}
	var a float64
	for i := range poly {
		j := (i + 1) % len(poly)
		a += poly[i].x*poly[j].y - poly[j].x*poly[i].y
	}
	return a / 2
}

// eoInside 用射线法判断点 q 是否在多边形 poly 内。
func eoInside(q eoPt, poly []eoPt) bool {
	if len(poly) < 3 {
		return false
	}
	inside := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		a, b := poly[i], poly[j]
		if (a.y > q.y) != (b.y > q.y) {
			x := (b.x-a.x)*(q.y-a.y)/(b.y-a.y) + a.x
			if q.x < x {
				inside = !inside
			}
		}
	}
	return inside
}

// eoCross 是向量 (a-o)x(b-o) 的 z 分量。
func eoCross(o, a, b eoPt) float64 {
	return (a.x-o.x)*(b.y-o.y) - (a.y-o.y)*(b.x-o.x)
}

// eoEdgesCross 判断两条线段是否真正穿越（共享端点、共线接触都不算）。
func eoEdgesCross(p1, p2, q1, q2 eoPt) bool {
	d1 := eoCross(p1, p2, q1)
	d2 := eoCross(p1, p2, q2)
	d3 := eoCross(q1, q2, p1)
	d4 := eoCross(q1, q2, p2)
	return ((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) &&
		((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0))
}

// eoSelfIntersect 检查单条闭合折线是否自交（跳过共享端点的相邻边）。
func eoSelfIntersect(poly []eoPt) bool {
	n := len(poly)
	if n < 4 {
		return false
	}
	for i := 0; i < n; i++ {
		a, b := poly[i], poly[(i+1)%n]
		for j := i + 2; j < n; j++ {
			if i == 0 && j == n-1 {
				continue // 首尾相邻边
			}
			if eoEdgesCross(a, b, poly[j], poly[(j+1)%n]) {
				return true
			}
		}
	}
	return false
}

// eoPolysCross 检查两条子路径边界是否互相穿越。若穿越，说明是「部分重叠」
// 而非严格嵌套 / 完全独立，方向交替 + nonzero 与 evenodd 不再等价。
func eoPolysCross(a, b []eoPt) bool {
	na, nb := len(a), len(b)
	if na < 3 || nb < 3 {
		return false
	}
	// 包围盒快速排除。
	minMax := func(p []eoPt) (float64, float64, float64, float64) {
		x0, y0, x1, y1 := p[0].x, p[0].y, p[0].x, p[0].y
		for _, q := range p[1:] {
			x0 = math.Min(x0, q.x)
			y0 = math.Min(y0, q.y)
			x1 = math.Max(x1, q.x)
			y1 = math.Max(y1, q.y)
		}
		return x0, y0, x1, y1
	}
	ax0, ay0, ax1, ay1 := minMax(a)
	bx0, by0, bx1, by1 := minMax(b)
	if ax1 < bx0 || bx1 < ax0 || ay1 < by0 || by1 < ay0 {
		return false
	}
	for i := 0; i < na; i++ {
		p1, p2 := a[i], a[(i+1)%na]
		for j := 0; j < nb; j++ {
			if eoEdgesCross(p1, p2, b[j], b[(j+1)%nb]) {
				return true
			}
		}
	}
	return false
}

// eoConvertible 判定是否满足「严格嵌套或完全独立」前提。自交、部分重叠、
// 以及点数过大的路径一律放弃转换，回退到原来的 even-odd 光栅化，宁可保留
// 已知缺陷也不引入新的错误填充。
func eoConvertible(polys [][]eoPt) bool {
	const maxPoints = 512
	total := 0
	for _, poly := range polys {
		total += len(poly)
	}
	if total > maxPoints {
		return false
	}
	for i, poly := range polys {
		if eoSelfIntersect(poly) {
			return false
		}
		for j := i + 1; j < len(polys); j++ {
			if eoPolysCross(poly, polys[j]) {
				return false
			}
		}
	}
	return true
}

// eoDepth 计算每条子路径的嵌套深度：面积更大的、且包含其代表点的兄弟数。
// 与修正 6 的 PoC 一致，代表点取折线中位点。
func eoDepth(polys [][]eoPt, areas []float64) []int {
	depths := make([]int, len(polys))
	for i := range polys {
		if len(polys[i]) == 0 {
			continue
		}
		q := polys[i][len(polys[i])/2]
		for j := range polys {
			if i == j || math.Abs(areas[j]) <= math.Abs(areas[i]) {
				continue
			}
			if eoInside(q, polys[j]) {
				depths[i]++
			}
		}
	}
	return depths
}

// eoReverseSub 反转一条子路径：段序列逆序、每段控制点倒序、终点换成原段
// 起点。原本由 Z 表达的隐含闭合段会被显式补成一条 Line，保证反转后仍闭合。
func eoReverseSub(s eoSub) eoSub {
	// 每段的起点（pts[k] 是第 k 段的起点）。
	starts := make([]fixed.Point26_6, len(s.segs))
	cur := s.start
	// 展开 Close 成显式 Line 到起点，便于统一处理。
	raw := make([]eoSeg, 0, len(s.segs))
	for k, seg := range s.segs {
		starts[k] = cur
		if seg.cmd == rasterx.PathClose {
			raw = append(raw, eoSeg{cmd: rasterx.PathLineTo, pts: []fixed.Point26_6{s.start}})
			cur = s.start
		} else {
			raw = append(raw, seg)
			cur = seg.pts[len(seg.pts)-1]
		}
	}

	// 新起点是原路径的终点；闭合路径下它恰好等于原起点。
	newStart := s.start
	if len(raw) > 0 {
		newStart = raw[len(raw)-1].pts[len(raw[len(raw)-1].pts)-1]
	}

	out := eoSub{start: newStart, closed: s.closed}
	for k := len(raw) - 1; k >= 0; k-- {
		seg := raw[k]
		end := starts[k] // 反转后这段的终点 = 原段的起点
		switch seg.cmd {
		case rasterx.PathLineTo:
			out.segs = append(out.segs, eoSeg{
				cmd: rasterx.PathLineTo,
				pts: []fixed.Point26_6{end},
			})
		case rasterx.PathQuadTo: // [ctrl, end] -> [ctrl, origStart]
			out.segs = append(out.segs, eoSeg{
				cmd: rasterx.PathQuadTo,
				pts: []fixed.Point26_6{seg.pts[0], end},
			})
		case rasterx.PathCubicTo: // [c1, c2, end] -> [c2, c1, origStart]
			out.segs = append(out.segs, eoSeg{
				cmd: rasterx.PathCubicTo,
				pts: []fixed.Point26_6{seg.pts[1], seg.pts[0], end},
			})
		}
	}
	if s.closed {
		out.segs = append(out.segs, eoSeg{cmd: rasterx.PathClose})
	}
	return out
}

// eoBuildPath 从子路径列表重建扁平 rasterx.Path。
func eoBuildPath(subs []eoSub) rasterx.Path {
	var out rasterx.Path
	for _, s := range subs {
		out.Start(s.start)
		for _, seg := range s.segs {
			switch seg.cmd {
			case rasterx.PathLineTo:
				out.Line(seg.pts[0])
			case rasterx.PathQuadTo:
				out.QuadBezier(seg.pts[0], seg.pts[1])
			case rasterx.PathCubicTo:
				out.CubeBezier(seg.pts[0], seg.pts[1], seg.pts[2])
			case rasterx.PathClose:
				out.Stop(true)
			}
		}
	}
	return out
}

// convertEvenOddToNonZero 把多子路径 Path 调整为「方向按嵌套深度交替」，
// 使其用 nonzero 填充时与 evenodd 等价。返回 false 表示不适用，调用方应
// 保持原来的 evenodd 行为。
func convertEvenOddToNonZero(p rasterx.Path) (rasterx.Path, bool) {
	subs := splitEOSubpaths(p)
	if len(subs) < 2 {
		return nil, false
	}

	polys := make([][]eoPt, len(subs))
	areas := make([]float64, len(subs))
	for i, s := range subs {
		polys[i] = eoSamplePoly(s)
		areas[i] = eoSignedArea(polys[i])
	}
	if !eoConvertible(polys) {
		return nil, false
	}
	depths := eoDepth(polys, areas)

	// 基准方向取最外层（depth=0）面积最大的子路径符号。
	base := 1.0
	best := -1.0
	found := false
	for i, d := range depths {
		if d == 0 && math.Abs(areas[i]) > best {
			best = math.Abs(areas[i])
			if areas[i] < 0 {
				base = -1.0
			} else {
				base = 1.0
			}
			found = true
		}
	}
	if !found {
		return nil, false
	}

	// 面积退化（零）或深度误判的子路径不参与反转判定，整体放弃转换更安全。
	for _, a := range areas {
		if a == 0 {
			return nil, false
		}
	}

	for i := range subs {
		target := base
		if depths[i]%2 == 1 {
			target = -base
		}
		if (areas[i] > 0) != (target > 0) {
			subs[i] = eoReverseSub(subs[i])
		}
	}
	return eoBuildPath(subs), true
}

// evenOddPathEligible 判断是否满足转换的保守前提：
//   - d 里不含 A/a/T/t 指令（PoC 明确跳过这类，避免弧线/平滑指令的
//     等价性尚未验证）；
//   - 至少两个子路径（单子路径的 evenodd 与 nonzero 等价，无需转换）。
func evenOddPathEligible(d string, p rasterx.Path) bool {
	for i := 0; i < len(d); i++ {
		switch d[i] {
		case 'A', 'a', 'T', 't':
			return false
		}
	}
	moves := 0
	for i := 0; i < len(p); {
		switch rasterx.PathCommand(p[i]) {
		case rasterx.PathMoveTo:
			moves++
			i += 3
		case rasterx.PathLineTo:
			i += 3
		case rasterx.PathQuadTo:
			i += 5
		case rasterx.PathCubicTo:
			i += 7
		default:
			i++
		}
	}
	return moves >= 2
}
