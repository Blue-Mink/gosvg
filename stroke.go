package gosvg

import (
	"image"
	"image/color"
	"math"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
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
func strokeCapsules(img *image.NRGBA, path rasterx.Path, matrix rasterx.Matrix2D, halfWidth float64, c color.NRGBA, tol float64) {
	subs := flattenPath(path, matrix, tol)
	if len(subs) == 0 {
		return
	}

	b := img.Bounds()
	scanner := rasterx.NewScannerGV(b.Dx(), b.Dy(), img, b)
	filler := rasterx.NewFiller(b.Dx(), b.Dy(), scanner)
	filler.SetWinding(true) // non-zero

	for _, pts := range subs {
		if len(pts) == 1 {
			// 仅 MoveTo 的子路径不产生绘制（SVG 规范：zero-length subpath
			// 只有含线段命令时才渲染 cap）。
			continue
		}
		for i := 0; i < len(pts)-1; i++ {
			a, bpt := pts[i], pts[i+1]
			if math.Hypot(bpt.X-a.X, bpt.Y-a.Y) < 1e-9 {
				addCircle(filler, bpt, halfWidth)
				continue
			}
			addCapsule(filler, a, bpt, halfWidth)
		}
	}

	filler.SetColor(c)
	filler.Draw()
}

// addCapsule 生成线段 [a,b] + 半宽 u 的胶囊体轮廓（round cap）。
// 两端半圆分别绕「前进方向」（b 端）与「后退方向」（a 端），保证圆帽正确外扩。
func addCapsule(f *rasterx.Filler, a, b Point, u float64) {
	dx := b.X - a.X
	dy := b.Y - a.Y
	l := math.Hypot(dx, dy)
	if l < 1e-9 {
		addCircle(f, a, u)
		return
	}
	nx := -dy / l * u // 法线
	ny := dx / l * u
	tx := dx / l * u // 切线（前进方向）
	ty := dy / l * u

	f.Start(fpt(a.X+nx, a.Y+ny))
	f.Line(fpt(b.X+nx, b.Y+ny))
	addHalfCircle(f, b, nx, ny, tx, ty, u) // b 端：+n → -n，绕前进方向
	f.Line(fpt(a.X-nx, a.Y-ny))
	addHalfCircle(f, a, -nx, -ny, -tx, -ty, u) // a 端：-n → +n，绕后退方向
	f.Stop(true)
}

// addHalfCircle 从 c+(nx,ny) 到 c-(nx,ny) 的半圆，绕 c+(tx,ty) 方向。
// 用两个 cubic bezier 精确近似（每个 90°，k=4/3·tan(π/8)），误差 ~0.0003·u。
func addHalfCircle(f *rasterx.Filler, c Point, nx, ny, tx, ty, u float64) {
	const k = 0.552284749831
	// 第一个 90°：n 方向 → t 方向
	c1 := Point{c.X + nx + k*tx, c.Y + ny + k*ty}
	p1 := Point{c.X + tx, c.Y + ty}
	c2 := Point{p1.X + k*nx, p1.Y + k*ny}
	f.CubeBezier(fpt(c1.X, c1.Y), fpt(c2.X, c2.Y), fpt(p1.X, p1.Y))
	// 第二个 90°：t 方向 → -n 方向
	c3 := Point{p1.X - k*nx, p1.Y - k*ny}
	p2 := Point{c.X - nx, c.Y - ny}
	c4 := Point{p2.X + k*tx, p2.Y + k*ty}
	f.CubeBezier(fpt(c3.X, c3.Y), fpt(c4.X, c4.Y), fpt(p2.X, p2.Y))
}

// addCircle 生成一个圆点（退化线段的 stroke）。
// 方向必须与 addCapsule 一致（视觉逆时针，y 向下坐标系），否则与相邻
// 胶囊体重叠处 non-zero winding 会抵消镂空。
func addCircle(f *rasterx.Filler, c Point, u float64) {
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
			f.Start(fpt(p0.X, p0.Y))
			first = false
		}
		f.CubeBezier(fpt(c1.X, c1.Y), fpt(c2.X, c2.Y), fpt(p1.X, p1.Y))
	}
	f.Stop(true)
}
