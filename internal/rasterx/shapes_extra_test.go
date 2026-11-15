package rasterx

import (
	"image"
	"image/color"
	"math"
	"testing"

	"golang.org/x/image/math/fixed"
)

// 本文件补 internal/rasterx/shapes.go / matrix.go / stroke.go 里此前 0% 的
// 形状构造与 Adder 适配覆盖。用 Path 作为 Adder 收集命令，断言「形状性质」
// （闭合、命令种类、坐标范围、变换效果），不写死具体顶点数或像素。

// countCmd 统计 Path 里某类命令 token 的出现次数（按命令长度步进，避免把坐标误当 token）。
func countCmd(p Path, want PathCommand) int {
	n := 0
	for i := 0; i < len(p); {
		c := PathCommand(p[i])
		if c == want {
			n++
		}
		switch c {
		case PathMoveTo, PathLineTo:
			i += 3
		case PathQuadTo:
			i += 5
		case PathCubicTo:
			i += 7
		case PathClose:
			i++
		default:
			return n // 不合法 token，交给调用方的其他断言报错
		}
	}
	return n
}

func pathBounds(p Path) (minX, minY, maxX, maxY fixed.Int26_6, ok bool) {
	first := true
	for i := 0; i < len(p); {
		c := PathCommand(p[i])
		var pts []fixed.Int26_6
		switch c {
		case PathMoveTo, PathLineTo:
			pts = p[i+1 : i+3]
			i += 3
		case PathQuadTo:
			pts = p[i+1 : i+5]
			i += 5
		case PathCubicTo:
			pts = p[i+1 : i+7]
			i += 7
		case PathClose:
			i++
			continue
		default:
			return 0, 0, 0, 0, false
		}
		for j := 0; j < len(pts); j += 2 {
			x, y := pts[j], pts[j+1]
			if first {
				minX, minY, maxX, maxY, first = x, y, x, y, false
				continue
			}
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	return minX, minY, maxX, maxY, !first
}

// ---------- ToFixedP ----------

func TestToFixedP(t *testing.T) {
	p := ToFixedP(2.5, -3.25)
	if p.X != fixed.Int26_6(2.5*64) || p.Y != fixed.Int26_6(-3.25*64) {
		t.Errorf("ToFixedP = %v, want (160,-208) in fixed units", p)
	}
	if got := ToFixedP(0, 0); got != (fixed.Point26_6{}) {
		t.Errorf("ToFixedP(0,0) = %v", got)
	}
}

// ---------- 形状构造 ----------

func TestAddCircleProducesClosedBezierPath(t *testing.T) {
	var p Path
	AddCircle(10, 10, 5, &p)
	if len(p) == 0 {
		t.Fatal("AddCircle 未产生任何路径命令")
	}
	if countCmd(p, PathMoveTo) != 1 {
		t.Errorf("圆应只有一个起点, got %d", countCmd(p, PathMoveTo))
	}
	if countCmd(p, PathCubicTo) < 4 {
		t.Errorf("整圆至少应有多段三次贝塞尔, got %d", countCmd(p, PathCubicTo))
	}
	if countCmd(p, PathClose) != 1 {
		t.Error("整圆应闭合 (PathClose)")
	}
	// 所有控制点应落在 [cx-r-ε, cx+r+ε] 包围盒内（贝塞尔控制点可略微外扩）
	minX, minY, maxX, maxY, ok := pathBounds(p)
	if !ok {
		t.Fatal("AddCircle 产生了无法解析的路径")
	}
	const eps = fixed.Int26_6(64) // 1 用户单位容差
	lo, hi := fixed.I(5)-eps, fixed.I(15)+eps
	if minX < lo || maxX > hi || minY < lo || maxY > hi {
		t.Errorf("圆控制点越界: x[%v,%v] y[%v,%v], want within [%v,%v]", minX, maxX, minY, maxY, lo, hi)
	}
}

func TestAddEllipseRespectsRadii(t *testing.T) {
	var p Path
	AddEllipse(0, 0, 8, 3, 0, &p) // 扁椭圆，rx>ry
	if countCmd(p, PathClose) != 1 || countCmd(p, PathCubicTo) < 4 {
		t.Fatalf("椭圆应为多段三次贝塞尔且闭合: %s", p.ToSVGPath())
	}
	minX, minY, maxX, maxY, _ := pathBounds(p)
	// 未旋转时 x 跨度应明显大于 y 跨度
	if (maxX - minX) <= (maxY - minY) {
		t.Errorf("rx=8,ry=3 的椭圆 x 跨度应更大: dx=%v dy=%v", maxX-minX, maxY-minY)
	}
}

func TestAddRectIdentityRotation(t *testing.T) {
	var p Path
	AddRect(2, 3, 12, 8, 0, &p) // rot=0 → MatrixAdder 用单位矩阵
	if countCmd(p, PathMoveTo) != 1 || countCmd(p, PathLineTo) != 3 || countCmd(p, PathClose) != 1 {
		t.Errorf("矩形应有 1 Start + 3 Line + 1 Close: %s", p.ToSVGPath())
	}
	minX, minY, maxX, maxY, ok := pathBounds(p)
	if !ok {
		t.Fatal("AddRect 路径无法解析")
	}
	if minX != fixed.I(2) || minY != fixed.I(3) || maxX != fixed.I(12) || maxY != fixed.I(8) {
		t.Errorf("rot=0 矩形顶点应原样保留: (%v,%v)-(%v,%v)", minX, minY, maxX, maxY)
	}
}

func TestAddRectRotationMovesVertices(t *testing.T) {
	var p0, p90 Path
	AddRect(0, 0, 10, 10, 0, &p0)
	AddRect(0, 0, 10, 10, 90, &p90)
	// 旋转 90° 后顶点集合应变化（正方形绕中心旋转后包围盒不变，但顶点坐标不同）
	if p0.ToSVGPath() == p90.ToSVGPath() {
		t.Error("AddRect 的 rot 参数未生效")
	}
	// 正方形绕自身中心旋转 90°，包围盒应仍近似一致
	min0X, _, max0X, _, _ := pathBounds(p0)
	min9X, _, max9X, _, _ := pathBounds(p90)
	if fapprox(float64(min9X-min0X), 0, 128) && fapprox(float64(max9X-max0X), 0, 128) == false {
		t.Errorf("绕中心旋转 90° 包围盒应基本不变: before[%v,%v] after[%v,%v]", min0X, max0X, min9X, max9X)
	}
}

func TestAddRoundRectDegeneratesToRect(t *testing.T) {
	var plain, degenerate Path
	AddRect(0, 0, 20, 10, 0, &plain)
	// rx<=0 时 AddRoundRect 直接退化成 AddRect（不看 gf）
	AddRoundRect(0, 0, 20, 10, 0, 4, 0, nil, &degenerate)
	if plain.ToSVGPath() != degenerate.ToSVGPath() {
		t.Errorf("rx<=0 应完全等同 AddRect:\n plain=%s\n got  =%s", plain.ToSVGPath(), degenerate.ToSVGPath())
	}
}

func TestAddRoundRectProducesCornerCurves(t *testing.T) {
	var p Path
	AddRoundRect(0, 0, 20, 20, 4, 4, 0, nil, &p) // gf=nil → FlatGap（直线倒角）
	if len(p) == 0 || countCmd(p, PathClose) != 1 {
		t.Fatal("圆角矩形应产生闭合路径")
	}
	if countCmd(p, PathLineTo) < 4 {
		t.Errorf("四条直边应至少产生 4 个 LineTo, got %d", countCmd(p, PathLineTo))
	}
	// 与同尺寸直角矩形不同
	var rect Path
	AddRect(0, 0, 20, 20, 0, &rect)
	if p.ToSVGPath() == rect.ToSVGPath() {
		t.Error("圆角矩形不应与普通矩形完全相同")
	}
	// rx 超过半宽时应被夹到 w/2 而不 panic / 不产生 NaN
	var clipped Path
	AddRoundRect(0, 0, 10, 10, 100, 100, 0, RoundGap, &clipped)
	if _, _, _, _, ok := pathBounds(clipped); !ok {
		t.Error("超大圆角半径产生了无法解析的路径")
	}
}

// ---------- 椭圆参数化辅助函数 ----------

func TestEllipseParametric(t *testing.T) {
	sin0, cos1 := math.Sin(0), math.Cos(0)
	// eta=0, 无旋转, a=8,b=3, 中心 (0,0) → 点 (8,0)
	px, py := ellipsePointAt(8, 3, sin0, cos1, 0, 0, 0)
	if !fapprox(px, 8, 1e-9) || !fapprox(py, 0, 1e-9) {
		t.Errorf("ellipsePointAt(eta=0) = (%.3f,%.3f), want (8,0)", px, py)
	}
	// 切向量在 eta=0 处应垂直于 x 轴（指向 +y）
	dx, dy := ellipsePrime(8, 3, sin0, cos1, 0, 0, 0)
	if !fapprox(dx, 0, 1e-9) || dy <= 0 {
		t.Errorf("ellipsePrime(eta=0) = (%.3f,%.3f), want (0, +)", dx, dy)
	}
	// 半径平方恒等式：点 (a cosη, b sinη) 满足 (x/a)²+(y/b)²=1
	eta := 0.7
	x, y := ellipsePointAt(8, 3, sin0, cos1, eta, 0, 0)
	if got := (x/8)*(x/8) + (y/3)*(y/3); !fapprox(got, 1, 1e-9) {
		t.Errorf("椭圆参数点不满足单位圆方程: %.6f", got)
	}
	// 旋转 90°（sinTheta=1, cosTheta=0）时点应落在 y 轴
	rx, ry := ellipsePointAt(8, 3, 1, 0, 0, 0, 0)
	if !fapprox(rx, 0, 1e-9) || !fapprox(ry, 8, 1e-9) {
		t.Errorf("旋转 90° 后 eta=0 点 = (%.3f,%.3f), want (0,8)", rx, ry)
	}
}

func TestFindEllipseCenterSymmetry(t *testing.T) {
	// 端点 (-3,0)→(3,0)、rx=ry=5、无旋转：圆心必在弦的垂直平分线 x=0 上，
	// 且到端点的距离为 5 → y=±4。sweep/smallArc 的不同组合只决定取哪一侧，
	// 两组对偶组合应互为镜像（这是 SVG 椭圆弧求心的数学性质，不是 (0,0)）。
	type result struct{ cx, cy float64 }
	var rs [4]result
	combos := [][2]bool{{true, true}, {false, false}, {true, false}, {false, true}}
	for i, combo := range combos {
		ra, rb := 5.0, 5.0
		cx, cy := FindEllipseCenter(&ra, &rb, 0, -3, 0, 3, 0, combo[0], combo[1])
		rs[i] = result{cx, cy}
		if !fapprox(cx, 0, 1e-9) {
			t.Errorf("sweep=%v smallArc=%v: cx=%.3f, want 0（垂直平分线）", combo[0], combo[1], cx)
		}
		if d := math.Hypot(cx-(-3), cy-0); !fapprox(d, 5, 1e-9) {
			t.Errorf("sweep=%v smallArc=%v: 圆心到起点距离=%.3f, want 5", combo[0], combo[1], d)
		}
	}
	if !fapprox(rs[0].cy, -rs[2].cy, 1e-9) || !fapprox(rs[1].cy, -rs[3].cy, 1e-9) {
		t.Errorf("对偶 sweep/smallArc 组合的圆心应互为 y 镜像: %+v", rs)
	}
	// 半长轴小于半弦长时必须原地放大半径以容纳弦
	ra, rb := 2.0, 2.0
	FindEllipseCenter(&ra, &rb, 0, -5, 0, 5, 0, true, true)
	if ra < 5 || rb < 5 {
		t.Errorf("半径应被放大以容纳 10 单位弦长: ra=%.3f rb=%.3f", ra, rb)
	}
}

func TestAddArcReturnsLastPoint(t *testing.T) {
	var p Path
	// points = {rx, ry, rotDeg, largeArc, sweep, endX, endY}
	pts := []float64{5, 5, 0, 0, 0, 10, 0}
	lx, ly := AddArc(pts, 5, 0, 0, 0, &p)
	if !fapprox(lx, 10, 1e-6) || !fapprox(ly, 0, 1e-6) {
		t.Errorf("AddArc 末点 = (%.3f,%.3f), want 显式终点 (10,0)", lx, ly)
	}
	if countCmd(p, PathCubicTo) == 0 {
		t.Error("AddArc 应产生至少一段三次贝塞尔")
	}
}

// ---------- MatrixAdder ----------

func TestMatrixAdderTransformsPoints(t *testing.T) {
	var out Path
	m := &MatrixAdder{Adder: &out, M: Identity.Translate(10, 20)}
	m.Start(fp(1, 2))
	m.Line(fp(3, 4))
	m.QuadBezier(fp(5, 6), fp(7, 8))
	m.CubeBezier(fp(9, 10), fp(11, 12), fp(13, 14))
	// 每个点都应 +（10,20）用户单位
	if out[1] != fixed.I(11) || out[2] != fixed.I(22) {
		t.Errorf("MatrixAdder.Start 未平移: (%v,%v)", out[1], out[2])
	}
	if out[4] != fixed.I(13) || out[5] != fixed.I(24) {
		t.Errorf("MatrixAdder.Line 未平移: (%v,%v)", out[4], out[5])
	}
	// 曲线控制点也须平移
	if out[7] != fixed.I(15) || out[8] != fixed.I(26) {
		t.Errorf("MatrixAdder.QuadBezier 控制点未平移: (%v,%v)", out[7], out[8])
	}
	if out[12] != fixed.I(19) || out[13] != fixed.I(30) {
		t.Errorf("MatrixAdder.CubeBezier 控制点未平移: (%v,%v)", out[12], out[13])
	}
}

func TestMatrixAdderReset(t *testing.T) {
	var out Path
	m := &MatrixAdder{Adder: &out, M: Identity.Translate(99, 99)}
	m.Reset()
	if m.M != Identity {
		t.Fatalf("Reset 后 M = %+v, want Identity", m.M)
	}
	m.Start(fp(3, 4))
	if out[1] != fixed.I(3) || out[2] != fixed.I(4) {
		t.Errorf("Reset 后点位不应再被平移: (%v,%v)", out[1], out[2])
	}
}

// ---------- Stroker 构造 / cap / gap ----------

func TestNewStrokerDefaults(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	sc := NewScannerGV(32, 32, img, img.Bounds())
	sc.SetColor(color.NRGBA{0, 0, 0, 255})
	s := NewStroker(32, 32, sc)
	if s.Scanner == nil {
		t.Fatal("NewStroker 未绑定 Scanner")
	}
	if s.u != fixed.I(1)/2 {
		t.Errorf("默认半宽 u = %v, want %v (width=1<<6 /2)", s.u, fixed.I(1)/2)
	}
	if s.JoinMode != MiterClip {
		t.Errorf("构造末尾 SetStroke 应把 JoinMode 设为 MiterClip, got %v", s.JoinMode)
	}
	if s.CapL == nil || s.CapT == nil || s.JoinGap == nil {
		t.Error("cap/gap 回调不应为 nil（SetStroke 有默认值回退）")
	}
}

func TestGapToCapAndCapFuncs(t *testing.T) {
	a := fp(10, 10)
	eNorm := fp(2, 0) // 端点法向
	for _, tc := range []struct {
		name string
		gf   GapFunc
	}{
		{"FlatGap", FlatGap},
		{"RoundGap", RoundGap},
		{"QuadraticGap", QuadraticGap},
		{"CubicGap", CubicGap},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p Path
			GapToCap(&p, a, eNorm, tc.gf)
			if len(p) == 0 {
				t.Fatal("GapToCap 未产生路径")
			}
			// 必须以 a+eNorm 起、以 a-eNorm 收
			if p[1] != a.X+eNorm.X || p[2] != a.Y+eNorm.Y {
				t.Errorf("起点应为 a+eNorm: (%v,%v)", p[1], p[2])
			}
			// 末尾 LineTo 的坐标为 [token, x, y]，应等于 a-eNorm。
			if p[len(p)-3] != fixed.Int26_6(PathLineTo) ||
				p[len(p)-2] != a.X-eNorm.X || p[len(p)-1] != a.Y-eNorm.Y {
				t.Errorf("终点应为 a-eNorm: (%v,%v)", p[len(p)-2], p[len(p)-1])
			}
		})
	}
	for _, tc := range []struct {
		name string
		cf   CapFunc
	}{
		{"ButtCap", ButtCap},
		{"SquareCap", SquareCap},
		{"RoundCap", RoundCap},
		{"CubicCap", CubicCap},
		{"QuadraticCap", QuadraticCap},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p Path
			tc.cf(&p, a, eNorm)
			if len(p) == 0 {
				t.Fatalf("%s 未产生路径", tc.name)
			}
			if countCmd(p, PathMoveTo) != 1 {
				t.Errorf("%s 应只有一个 Start, got %d", tc.name, countCmd(p, PathMoveTo))
			}
		})
	}
}

func TestDasherCurveMethodsDoNotPanic(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	sc := NewScannerGV(32, 32, img, img.Bounds())
	sc.SetColor(color.NRGBA{0, 0, 0, 255})
	d := NewDasher(32, 32, sc)
	// Dasher 的 QuadBezier / CubeBezier 内部委托给 sgm（Stroker）；
	// 先用 SetStroke 配置合法参数，再从 Start 起调用，覆盖 0% 方法体。
	d.SetStroke(fixed.I(2), fixed.I(4), ButtCap, nil, FlatGap, MiterClip, nil, 0)
	d.Start(fp(4, 16))
	d.QuadBezier(fp(16, 2), fp(28, 16))
	d.Start(fp(4, 16))
	d.CubeBezier(fp(10, 2), fp(22, 30), fp(28, 16))
	// 只要不 panic 且能走到 Draw 即可（不写死像素）
	sc.Draw()
}
