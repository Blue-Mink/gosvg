package gosvg

import (
	"image"
	"math"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
	"golang.org/x/image/math/fixed"
)

// Point 是 float64 像素坐标。
type Point struct{ X, Y float64 }

// flattenPath 把 rasterx.Path 展开成**子路径列表**（每个子路径一条折线；
// 子路径之间绝不连线），quad/cubic 用 de Casteljau 递归 flatten（弧高误差
// < tol 像素），并对每个点应用 matrix 变换。tol 以像素为单位。
func flattenPath(path rasterx.Path, matrix rasterx.Matrix2D, tol float64) [][]Point {
	var subpaths [][]Point
	var cur []Point
	var cpx, cpy float64 // 像素坐标当前点
	var subStartP Point  // 当前子路径起点（像素坐标）

	flush := func() {
		if len(cur) > 0 {
			subpaths = append(subpaths, cur)
			cur = nil
		}
	}

	for i := 0; i < len(path); {
		switch rasterx.PathCommand(path[i]) {
		case rasterx.PathMoveTo:
			flush()
			fx, fy := float64(path[i+1])/64, float64(path[i+2])/64
			px, py := matrix.Transform(fx, fy)
			cur = append(cur, Point{px, py})
			cpx, cpy = px, py
			subStartP = Point{px, py}
			i += 3
		case rasterx.PathLineTo:
			fx, fy := float64(path[i+1])/64, float64(path[i+2])/64
			px, py := matrix.Transform(fx, fy)
			cur = append(cur, Point{px, py})
			cpx, cpy = px, py
			i += 3
		case rasterx.PathQuadTo:
			bx, by := float64(path[i+1])/64, float64(path[i+2])/64
			ex, ey := float64(path[i+3])/64, float64(path[i+4])/64
			bpx, bpy := matrix.Transform(bx, by)
			epx, epy := matrix.Transform(ex, ey)
			flattenQuad(&cur, Point{cpx, cpy}, Point{bpx, bpy}, Point{epx, epy}, tol)
			cpx, cpy = epx, epy
			i += 5
		case rasterx.PathCubicTo:
			bx, by := float64(path[i+1])/64, float64(path[i+2])/64
			cx2, cy2 := float64(path[i+3])/64, float64(path[i+4])/64
			ex, ey := float64(path[i+5])/64, float64(path[i+6])/64
			bpx, bpy := matrix.Transform(bx, by)
			c2px, c2py := matrix.Transform(cx2, cy2)
			epx, epy := matrix.Transform(ex, ey)
			flattenCubic(&cur, Point{cpx, cpy}, Point{bpx, bpy}, Point{c2px, c2py}, Point{epx, epy}, tol)
			cpx, cpy = epx, epy
			i += 7
		case rasterx.PathClose:
			// 闭合：连回**当前子路径**起点（不是整个 path 的第一个点）。
			cur = append(cur, subStartP)
			cpx, cpy = subStartP.X, subStartP.Y
			i++
		default:
			i++
		}
	}
	flush()
	return subpaths
}

// flattenQuad 在像素坐标下递归 de Casteljau flatten 二次贝塞尔（tol 为像素）。
func flattenQuad(out *[]Point, p0, p1, p2 Point, tol float64) {
	d1 := distToSegment(p1, p0, p2)
	if d1 < tol {
		*out = append(*out, p2)
		return
	}
	p01 := mid(p0, p1)
	p12 := mid(p1, p2)
	p012 := mid(p01, p12)
	flattenQuad(out, p0, p01, p012, tol)
	flattenQuad(out, p012, p12, p2, tol)
}

// flattenCubic 在像素坐标下递归 de Casteljau flatten 三次贝塞尔（tol 为像素）。
func flattenCubic(out *[]Point, p0, p1, p2, p3 Point, tol float64) {
	d1 := distToSegment(p1, p0, p3)
	d2 := distToSegment(p2, p0, p3)
	if d1 < tol && d2 < tol {
		*out = append(*out, p3)
		return
	}
	p01 := mid(p0, p1)
	p12 := mid(p1, p2)
	p23 := mid(p2, p3)
	p012 := mid(p01, p12)
	p123 := mid(p12, p23)
	p0123 := mid(p012, p123)
	flattenCubic(out, p0, p01, p012, p0123, tol)
	flattenCubic(out, p0123, p123, p23, p3, tol)
}

func mid(a, b Point) Point { return Point{(a.X + b.X) / 2, (a.Y + b.Y) / 2} }

func distToSegment(p, a, b Point) float64 {
	dx := b.X - a.X
	dy := b.Y - a.Y
	l2 := dx*dx + dy*dy
	if l2 < 1e-12 {
		return math.Hypot(p.X-a.X, p.Y-a.Y)
	}
	t := ((p.X-a.X)*dx + (p.Y-a.Y)*dy) / l2
	t = math.Max(0, math.Min(1, t))
	px := a.X + t*dx
	py := a.Y + t*dy
	return math.Hypot(p.X-px, p.Y-py)
}

// strokeCapsules 用「胶囊体并集」渲染 round-cap + round-join 的 stroke。
//
// 原理：把路径 flatten 成折线后，每条线段独立生成一个胶囊体（偏移矩形 + 两端
// 半圆），全部用 non-zero winding 累积进 rasterx 的 Filler 再填充。相邻胶囊体
// 的端半圆在 join 处自然重叠成一个整圆，所以 round join 无需显式处理。
//
// 为什么不用 rasterx 的 Stroker：rasterx 是「整体偏移轮廓 + non-zero 填充」，
// 当曲线半径 ≈ 半宽（内偏移退化，如半径 1 的弧配 width 2 的描边）时内侧轮廓
// 自交，non-zero winding 填出错误的孔。胶囊体并集的每个胶囊体都是独立的简单
// 几何，重叠区域 winding 累加但非零，天然免疫自交退化——这正是 cairo 质量
// 的关键差异。
//
// 为什么填充器用 rasterx.Filler 而非 x/image/vector：两者是独立的光栅化实现，
// 与 cairosvg 的 AA 对齐程度不同（实测 fill 圆 rasterx=0.9986 / vector=0.9952）。
// stroke 与 fill 共用 rasterx 的扫描器，能把 stroke 的 AA 拉到与 fill 同一水平。
// ---- 精确 stroke 轮廓：偏移线 + 圆弧 join，单条连续边 → 解析级 AA ----
// （胶囊体并集的边缘被相邻胶囊体双重覆盖，coverage 饱和成硬边，AA 渐变被抹平；
//   精确轮廓的每条边是单条边链，像素 coverage 为真实部分值。）

func emitArcDir(f *rasterx.Filler, pt func(x, y float64) fixed.Point26_6, arcTol, cx, cy, a0, delta, r float64) {
	step := 2 * math.Acos(1-arcTol/r)
	if step <= 0 || step > 1 {
		step = 1
	}
	n := int(math.Ceil(math.Abs(delta)/step)) + 1
	for k := 1; k <= n; k++ {
		a := a0 + delta*float64(k)/float64(n)
		f.Line(pt(cx+r*math.Cos(a), cy+r*math.Sin(a)))
	}
}

func emitArcShort(f *rasterx.Filler, pt func(x, y float64) fixed.Point26_6, arcTol, cx, cy, a0, a1, r float64) {
	d := a1 - a0
	for d > math.Pi {
		d -= 2 * math.Pi
	}
	for d <= -math.Pi {
		d += 2 * math.Pi
	}
	emitArcDir(f, pt, arcTol, cx, cy, a0, d, r)
}

// lineIntersect 直线 p+t·d 与 q+s·e 求交。
func lineIntersect(px, py, dx, dy, qx, qy, ex, ey float64) (float64, float64, bool) {
	den := dx*ey - dy*ex
	if math.Abs(den) < 1e-12 {
		return 0, 0, false
	}
	t := ((qx-px)*ey - (qy-py)*ex) / den
	return px + t*dx, py + t*dy, true
}

type segDir struct{ tx, ty, nx, ny float64 }

func segOf(a, b Point, u float64) segDir {
	dx := b.X - a.X
	dy := b.Y - a.Y
	l := math.Hypot(dx, dy)
	return segDir{dx / l, dy / l, -dy / l * u, dx / l * u}
}

// emitJoin 处理顶点 pts[idx] 处 segs[prev]→segs[idx] 的 join。
func emitJoin(f *rasterx.Filler, pt func(x, y float64) fixed.Point26_6, arcTol float64, pts []Point, segs []segDir, idx, n int, u float64, join string, miterLimit float64) {
	prev := (idx - 1 + n) % n
	pv := segs[prev]
	s := segs[idx]
	P := pts[idx]
	cross := pv.tx*s.ty - pv.ty*s.tx
	dot := pv.tx*s.tx + pv.ty*s.ty
	aj := math.Atan2(pv.ny, pv.nx)
	ai := math.Atan2(s.ny, s.nx)
	q2x, q2y := P.X+s.nx, P.Y+s.ny
	switch {
	case math.Abs(cross) < 1e-12:
		if dot < 0 {
			if join == "round" {
				emitArcDir(f, pt, arcTol, P.X, P.Y, aj, -math.Pi, u)
			} else {
				f.Line(pt(q2x, q2y))
			}
		}
	default:
		X, Y, ok := lineIntersect(P.X+pv.nx, P.Y+pv.ny, pv.tx, pv.ty,
			P.X+s.nx, P.Y+s.ny, s.tx, s.ty)
		d := math.Hypot(X-P.X, Y-P.Y)
		if ok && d < u*(1-1e-9) { // 凹：内交点
			f.Line(pt(X, Y))
		} else {
			switch join {
			case "bevel":
				f.Line(pt(q2x, q2y))
			case "miter":
				// SVG: miterLength/strokeWidth > miterlimit 时 bevel。
				// miterLength=d（顶点到尖点），strokeWidth=2u，故阈值 d <= 2*miterLimit*u。
				if ok && d <= 2*miterLimit*u {
					f.Line(pt(X, Y))
				} else {
					f.Line(pt(q2x, q2y))
				}
			default:
				emitArcShort(f, pt, arcTol, P.X, P.Y, aj, ai, u)
			}
		}
	}
}

// emitOffsetChain 沿链生成左偏移边（含顶点 join）。
// closed=true 时为完整环（含首顶点 join，终点回到起点）。返回最后法线角。
func emitOffsetChain(f *rasterx.Filler, pt func(x, y float64) fixed.Point26_6, arcTol float64, pts []Point, u float64, closed bool, join string, miterLimit float64) float64 {
	n := len(pts)
	segN := n - 1
	if closed {
		segN = n
	}
	segs := make([]segDir, segN)
	for i := 0; i < segN; i++ {
		segs[i] = segOf(pts[i], pts[(i+1)%n], u)
	}
	lastAng := 0.0
	for i := 0; i < segN; i++ {
		s := segs[i]
		if i > 0 {
			emitJoin(f, pt, arcTol, pts, segs, i, segN, u, join, miterLimit)
		}
		b := pts[(i+1)%n]
		f.Line(pt(b.X+s.nx, b.Y+s.ny))
		lastAng = math.Atan2(s.ny, s.nx)
	}
	if closed {
		emitJoin(f, pt, arcTol, pts, segs, 0, segN, u, join, miterLimit)
	}
	return lastAng
}

// strokeExactPath 把一个子路径的精确 stroke 轮廓写入 filler。
func strokeExactPath(f *rasterx.Filler, pt func(x, y float64) fixed.Point26_6, arcTol float64, pts []Point, u float64, closed bool, cap, join string, miterLimit float64) {
	n := len(pts)
	if (closed && n < 3) || (!closed && n < 2) {
		if join == "round" {
			addCircle(f, pt, pts[0], u)
		}
		return
	}
	if closed {
		f.Start(pt(pts[0].X+segOf(pts[0], pts[1], u).nx, pts[0].Y+segOf(pts[0], pts[1], u).ny))
		emitOffsetChain(f, pt, arcTol, pts, u, true, join, miterLimit)
		f.Stop(true)
		rev := make([]Point, n)
		for i := range pts {
			rev[i] = pts[n-1-i]
		}
		f.Start(pt(rev[0].X+segOf(rev[0], rev[1], u).nx, rev[0].Y+segOf(rev[0], rev[1], u).ny))
		emitOffsetChain(f, pt, arcTol, rev, u, true, join, miterLimit)
		f.Stop(true)
		return
	}
	s0 := segOf(pts[0], pts[1], u)
	f.Start(pt(pts[0].X+s0.nx, pts[0].Y+s0.ny))
	emitOffsetChain(f, pt, arcTol, pts, u, false, join, miterLimit)
	sl := segOf(pts[n-2], pts[n-1], u)
	P0 := pts[0]
	Pl := pts[n-1]
	aN := math.Atan2(sl.ny, sl.nx)
	aN0 := math.Atan2(s0.ny, s0.nx)
	// ---- capEnd ----
	switch cap {
	case "square":
		f.Line(pt(Pl.X+sl.nx+sl.tx*u, Pl.Y+sl.ny+sl.ty*u))
		f.Line(pt(Pl.X-sl.nx+sl.tx*u, Pl.Y-sl.ny+sl.ty*u))
		f.Line(pt(Pl.X-sl.nx, Pl.Y-sl.ny))
	case "butt":
		f.Line(pt(Pl.X-sl.nx, Pl.Y-sl.ny))
	default: // round
		emitArcDir(f, pt, arcTol, Pl.X, Pl.Y, aN, -math.Pi, u)
	}
	rev := make([]Point, n)
	for i := range pts {
		rev[i] = pts[n-1-i]
	}
	emitOffsetChain(f, pt, arcTol, rev, u, false, join, miterLimit)
	// ---- cap0 ----
	switch cap {
	case "square":
		f.Line(pt(P0.X-s0.nx-s0.tx*u, P0.Y-s0.ny-s0.ty*u))
		f.Line(pt(P0.X+s0.nx-s0.tx*u, P0.Y+s0.ny-s0.ty*u))
		f.Line(pt(P0.X+s0.nx, P0.Y+s0.ny))
	case "butt":
		f.Line(pt(P0.X+s0.nx, P0.Y+s0.ny))
	default: // round
		emitArcDir(f, pt, arcTol, P0.X, P0.Y, aN0+math.Pi, -math.Pi, u)
	}
	f.Stop(true)
}

func segs0N(pts []Point, u float64) segDir { return segOf(pts[0], pts[1], u) }

func strokeCapsules(img *image.NRGBA, path rasterx.Path, matrix rasterx.Matrix2D, halfWidth float64, clr interface{}, tol float64, linecap, linejoin string, miterLimit float64, dasharray []float64, dashoffset float64) {
	// 用户空间 flatten（Identity），offset 在用户空间做，最终 Transform 到 device。
	subs := flattenPath(path, rasterx.Identity, tol)
	if len(subs) == 0 {
		return
	}

	b := img.Bounds()
	scanner := rasterx.NewScannerGV(b.Dx(), b.Dy(), img, b)
	filler := rasterx.NewFiller(b.Dx(), b.Dy(), scanner)
	filler.SetWinding(true) // non-zero
	pt := func(x, y float64) fixed.Point26_6 {
		tx, ty := matrix.Transform(x, y)
		return fpt(tx, ty)
	}
	dashes := normalizeDashes(dasharray)

	for _, pts := range subs {
		// 去除相邻重复点
		dedup := pts[:0]
		for i, p := range pts {
			if i == 0 || math.Hypot(p.X-dedup[len(dedup)-1].X, p.Y-dedup[len(dedup)-1].Y) > 1e-9 {
				dedup = append(dedup, p)
			}
		}
		if len(dedup) < 2 {
			if len(dedup) == 1 && linejoin == "round" {
				addCircle(filler, pt, dedup[0], halfWidth)
			}
			continue
		}
		closedP := math.Hypot(dedup[0].X-dedup[len(dedup)-1].X,
			dedup[0].Y-dedup[len(dedup)-1].Y) < 1e-6 && len(dedup) > 3
		if closedP {
			dedup = dedup[:len(dedup)-1]
		}

		if len(dashes) > 0 {
			// dash：闭合环展开成首尾相连的开放折线，切段后每段独立 stroke。
			dashPts := dedup
			if closedP {
				dashPts = make([]Point, 0, len(dedup)+1)
				dashPts = append(dashPts, dedup...)
				dashPts = append(dashPts, dedup[0])
			}
			for _, seg := range dashSegments(dashPts, dashoffset, dashes, closedP) {
				if len(seg) >= 2 {
					strokeExactPath(filler, pt, tol, seg, halfWidth, false, linecap, linejoin, miterLimit)
				}
			}
			continue
		}
		strokeExactPath(filler, pt, tol, dedup, halfWidth, closedP, linecap, linejoin, miterLimit)
	}

	filler.SetColor(clr)
	filler.Draw()
}

// addCircle 生成一个圆点（退化线段的 stroke）。
// 方向必须与 addCapsule 一致（视觉逆时针，y 向下坐标系），否则与相邻
// 胶囊体重叠处 non-zero winding 会抵消镂空。
func addCircle(f *rasterx.Filler, pt func(x, y float64) fixed.Point26_6, c Point, u float64) {
	const k = 0.552284749831
	// 4 个 90° 弧（视觉逆时针）：右 → 上 → 左 → 下。R(x,y)=(y,-x)。
	dirs := [4][2]float64{{1, 0}, {0, -1}, {-1, 0}, {0, 1}}
	first := true
	for _, d := range dirs {
		nx, ny := d[0], d[1]
		rx, ry := ny, -nx // R(d)
		p0 := Point{c.X + nx*u, c.Y + ny*u}
		p1 := Point{c.X + rx*u, c.Y + ry*u}
		c1 := Point{p0.X + k*rx*u, p0.Y + k*ry*u}
		c2 := Point{p1.X + k*nx*u, p1.Y + k*ny*u}
		if first {
			f.Start(pt(p0.X, p0.Y))
			first = false
		}
		f.CubeBezier(pt(c1.X, c1.Y), pt(c2.X, c2.Y), pt(p1.X, p1.Y))
	}
	f.Stop(true)
}
