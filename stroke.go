package gosvg

import (
	"image"
	"image/color"
	"math"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
	"golang.org/x/image/vector"
)

// Point 是 float64 像素坐标。
type Point struct{ X, Y float64 }

// flattenPath 把 rasterx.Path 展开成**子路径列表**（每个子路径一条折线；
// 子路径之间绝不连线），quad/cubic 用 de Casteljau 递归 flatten（弧高误差
// < tol 像素），并对每个点应用 matrix 变换。
func flattenPath(path rasterx.Path, matrix rasterx.Matrix2D, tol float64) [][]Point {
	var subpaths [][]Point
	var cur []Point
	var cx, cy float64
	var subStartU, subStartP Point // 当前子路径起点（用户坐标 / 像素坐标）

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
			cx, cy = fx, fy
			subStartU, subStartP = Point{fx, fy}, Point{px, py}
			i += 3
		case rasterx.PathLineTo:
			fx, fy := float64(path[i+1])/64, float64(path[i+2])/64
			px, py := matrix.Transform(fx, fy)
			cur = append(cur, Point{px, py})
			cx, cy = fx, fy
			i += 3
		case rasterx.PathQuadTo:
			bx, by := float64(path[i+1])/64, float64(path[i+2])/64
			ex, ey := float64(path[i+3])/64, float64(path[i+4])/64
			flattenQuad(&cur, matrix, Point{cx, cy}, Point{bx, by}, Point{ex, ey}, tol)
			cx, cy = ex, ey
			i += 5
		case rasterx.PathCubicTo:
			bx, by := float64(path[i+1])/64, float64(path[i+2])/64
			cx2, cy2 := float64(path[i+3])/64, float64(path[i+4])/64
			ex, ey := float64(path[i+5])/64, float64(path[i+6])/64
			flattenCubic(&cur, matrix, Point{cx, cy}, Point{bx, by}, Point{cx2, cy2}, Point{ex, ey}, tol)
			cx, cy = ex, ey
			i += 7
		case rasterx.PathClose:
			// 闭合：连回**当前子路径**起点（不是整个 path 的第一个点）。
			cur = append(cur, subStartP)
			cx, cy = subStartU.X, subStartU.Y
			i++
		default:
			i++
		}
	}
	flush()
	return subpaths
}

func xform(m rasterx.Matrix2D, p Point) Point {
	px, py := m.Transform(p.X, p.Y)
	return Point{px, py}
}

func flattenQuad(out *[]Point, m rasterx.Matrix2D, p0, p1, p2 Point, tol float64) {
	d1 := distToSegment(p1, p0, p2)
	if d1 < tol {
		*out = append(*out, xform(m, p2))
		return
	}
	p01 := mid(p0, p1)
	p12 := mid(p1, p2)
	p012 := mid(p01, p12)
	flattenQuad(out, m, p0, p01, p012, tol)
	flattenQuad(out, m, p012, p12, p2, tol)
}

func flattenCubic(out *[]Point, m rasterx.Matrix2D, p0, p1, p2, p3 Point, tol float64) {
	d1 := distToSegment(p1, p0, p3)
	d2 := distToSegment(p2, p0, p3)
	if d1 < tol && d2 < tol {
		*out = append(*out, xform(m, p3))
		return
	}
	p01 := mid(p0, p1)
	p12 := mid(p1, p2)
	p23 := mid(p2, p3)
	p012 := mid(p01, p12)
	p123 := mid(p12, p23)
	p0123 := mid(p012, p123)
	flattenCubic(out, m, p0, p01, p012, p0123, tol)
	flattenCubic(out, m, p0123, p123, p23, p3, tol)
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
// 半圆），全部用 non-zero winding 累积进一个 Rasterizer 再填充。相邻胶囊体的
// 端半圆在 join 处自然重叠成一个整圆，所以 round join 无需显式处理。
//
// 为什么不用 rasterx 的 Stroker：rasterx 是「整体偏移轮廓 + non-zero 填充」，
// 当曲线半径 ≈ 半宽（内偏移退化，如半径 1 的弧配 width 2 的描边）时内侧轮廓
// 自交，non-zero winding 填出错误的孔。胶囊体并集的每个胶囊体都是独立的简单
// 几何，重叠区域 winding 累加但非零，天然免疫自交退化——这正是 cairo 质量
// 的关键差异。
//
// 已知边界（扇谷效应）：凸曲线外侧由相邻端半圆相交构成，交点比真实偏移弧
// 内缩约 R·tol/u 像素（R 曲率半径、tol flatten 容差、u 半宽）。tol=0.01 时
// 谷深 <0.05px，亚像素级；再降 tol 收益递减（AA 边缘的算法差异成为主导）。
func strokeCapsules(img *image.NRGBA, path rasterx.Path, matrix rasterx.Matrix2D, halfWidth float64, c color.NRGBA, tol float64) {
	subs := flattenPath(path, matrix, tol)
	if len(subs) == 0 {
		return
	}

	b := img.Bounds()
	z := vector.NewRasterizer(b.Dx(), b.Dy())

	for _, pts := range subs {
		if len(pts) == 1 {
			// 仅 MoveTo 的子路径不产生绘制（SVG 规范：zero-length subpath
			// 只有含线段命令时才渲染 cap）。
			continue
		}
		for i := 0; i < len(pts)-1; i++ {
			a, bpt := pts[i], pts[i+1]
			if math.Hypot(bpt.X-a.X, bpt.Y-a.Y) < 1e-9 {
				addCircle(z, bpt, halfWidth)
				continue
			}
			addCapsule(z, a, bpt, halfWidth)
		}
	}

	z.Draw(img, b, image.NewUniform(c), image.Point{})
}

// addCapsule 生成线段 [a,b] + 半宽 u 的胶囊体轮廓（round cap）。
// 两端半圆分别绕「前进方向」（b 端）与「后退方向」（a 端），保证圆帽正确外扩。
func addCapsule(z *vector.Rasterizer, a, b Point, u float64) {
	dx := b.X - a.X
	dy := b.Y - a.Y
	l := math.Hypot(dx, dy)
	if l < 1e-9 {
		addCircle(z, a, u)
		return
	}
	nx := -dy / l * u // 法线
	ny := dx / l * u
	tx := dx / l * u // 切线（前进方向）
	ty := dy / l * u

	z.MoveTo(float32(a.X+nx), float32(a.Y+ny))
	z.LineTo(float32(b.X+nx), float32(b.Y+ny))
	addHalfCircle(z, b, nx, ny, tx, ty, u) // b 端：+n → -n，绕前进方向
	z.LineTo(float32(a.X-nx), float32(a.Y-ny))
	addHalfCircle(z, a, -nx, -ny, -tx, -ty, u) // a 端：-n → +n，绕后退方向
	z.ClosePath()
}

// addHalfCircle 从 c+(nx,ny) 到 c-(nx,ny) 的半圆，绕 c+(tx,ty) 方向。
func addHalfCircle(z *vector.Rasterizer, c Point, nx, ny, tx, ty, u float64) {
	const segs = 48
	for i := 1; i <= segs; i++ {
		th := math.Pi * float64(i) / segs
		px := c.X + math.Cos(th)*nx + math.Sin(th)*tx
		py := c.Y + math.Cos(th)*ny + math.Sin(th)*ty
		z.LineTo(float32(px), float32(py))
	}
}

// addCircle 生成一个圆点（退化线段的 stroke）。
// 方向必须与 addCapsule 一致（视觉逆时针，y 向下坐标系），否则与相邻
// 胶囊体重叠处 non-zero winding 会抵消镂空。
func addCircle(z *vector.Rasterizer, c Point, u float64) {
	const segs = 48
	for i := segs; i >= 0; i-- {
		th := 2 * math.Pi * float64(i) / segs
		px := c.X + math.Cos(th)*u
		py := c.Y + math.Sin(th)*u
		if i == segs {
			z.MoveTo(float32(px), float32(py))
		} else {
			z.LineTo(float32(px), float32(py))
		}
	}
	z.ClosePath()
}
