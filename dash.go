package gosvg

import (
	"math"
	"strings"
)

// parseDashArray 解析 stroke-dasharray 属性。返回 (dashes, ok)。
// ok=false 表示 "none" 或解析失败（按无 dash 处理）。
func parseDashArray(s string) ([]float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "none" {
		return nil, false
	}
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	out := make([]float64, 0, len(fields))
	for _, f := range fields {
		v, ok := parseFloat(f)
		if !ok {
			return nil, false
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// normalizeDashes 规范化 dash 数组：奇数长度复制成偶数（SVG 语义
// "d1 d2 d3" ≡ "d1 d2 d3 d1 d2 d3"）。含负值或全零 → 返回 nil（不 dash）。
func normalizeDashes(d []float64) []float64 {
	if len(d) == 0 {
		return nil
	}
	if len(d)%2 == 1 {
		d = append(append([]float64{}, d...), d...)
	}
	allZero := true
	for _, v := range d {
		if v < 0 {
			return nil
		}
		if v != 0 {
			allZero = false
		}
	}
	if allZero {
		return nil
	}
	return d
}

// dashSegments 把折线 pts 按 dash 模式切成多条开放子折线（每个 dash 一段）。
// closed 表示 pts 是一条"环"（首尾不相连、但几何上闭合）；闭合时若某个 dash
// 跨越了路径起点（offset 使 dash 从 0 之前开始），把首末两段合并成一段。
func dashSegments(pts []Point, offset float64, dashes []float64, closed bool) [][]Point {
	L := polylineLength(pts)
	if L <= 0 {
		return nil
	}
	ranges := dashRanges(offset, dashes, L)
	out := make([][]Point, 0, len(ranges))
	for _, r := range ranges {
		if seg := extractPolyline(pts, r[0], r[1]); len(seg) >= 2 {
			out = append(out, seg)
		}
	}
	if closed && len(out) >= 2 {
		first, last := out[0], out[len(out)-1]
		p0 := pts[0]
		if dist(first[0], p0) < 1e-6 && dist(last[len(last)-1], p0) < 1e-6 {
			merged := append(append([]Point{}, last...), first[1:]...)
			out = append(out[1:len(out)-1], merged)
		}
	}
	return out
}

// dashRanges 生成 [start,end] 弧长区间列表（dash 覆盖的部分）。
// dashes 已规范化（偶数长度、正值），offset 为 stroke-dashoffset。
func dashRanges(offset float64, dashes []float64, L float64) [][2]float64 {
	dTotal := 0.0
	for _, d := range dashes {
		dTotal += d
	}
	if dTotal <= 0 {
		return nil
	}
	off := math.Mod(offset, dTotal)
	if off < 0 {
		off += dTotal
	}
	// dashStart：i0 = off 所在的 dash 区间 index。
	i0 := 0
	for off >= dashes[i0] {
		off -= dashes[i0]
		i0++
		if i0 == len(dashes) {
			i0 = 0
		}
	}
	var ranges [][2]float64
	i := i0
	start := -off
	end := start + dashes[i]
	for start < L {
		if i%2 == 0 { // 偶数 index = dash
			s := math.Max(0, start)
			e := math.Min(L, end)
			if e > s {
				ranges = append(ranges, [2]float64{s, e})
			}
		}
		start = end
		i++
		if i == len(dashes) {
			i = 0
		}
		end = start + dashes[i]
	}
	return ranges
}

func polylineLength(pts []Point) float64 {
	var L float64
	for i := 1; i < len(pts); i++ {
		L += math.Hypot(pts[i].X-pts[i-1].X, pts[i].Y-pts[i-1].Y)
	}
	return L
}

// polylinePointAt 返回折线上弧长 t 处的点（t 越界时钳到端点）。
func polylinePointAt(pts []Point, t float64) Point {
	for i := 1; i < len(pts); i++ {
		seg := math.Hypot(pts[i].X-pts[i-1].X, pts[i].Y-pts[i-1].Y)
		if t <= seg {
			if seg <= 1e-12 {
				return pts[i]
			}
			f := t / seg
			return Point{pts[i-1].X + f*(pts[i].X-pts[i-1].X), pts[i-1].Y + f*(pts[i].Y-pts[i-1].Y)}
		}
		t -= seg
	}
	return pts[len(pts)-1]
}

// extractPolyline 提取折线上弧长 [a,b] 区间的子折线（含两端切点与内部顶点）。
func extractPolyline(pts []Point, a, b float64) []Point {
	out := []Point{polylinePointAt(pts, a)}
	cum := 0.0
	for i := 1; i < len(pts); i++ {
		cum += math.Hypot(pts[i].X-pts[i-1].X, pts[i].Y-pts[i-1].Y)
		if cum > a && cum < b {
			out = append(out, pts[i])
		}
	}
	endPt := polylinePointAt(pts, b)
	if last := out[len(out)-1]; dist(last, endPt) > 1e-9 {
		out = append(out, endPt)
	}
	return out
}

func dist(a, b Point) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}
