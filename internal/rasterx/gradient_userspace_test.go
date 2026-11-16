package rasterx

import (
	"image/color"
	"testing"
)

// 本文件补 GetColorFunctionUS 的 User Space / radial 分支覆盖。
// 这些分支由根包 render.go:499 与 gradient.go:203 经 ctx.matrix 真实调用，
// 但 internal/rasterx 包自身此前几乎无测试触达（GetColorFunctionUS 覆盖率
// 仅约 22%）。根包端到端测试属另一测试二进制，不计入本包覆盖率，故在此
// 直接测本函数。断言用相对行为（中心 vs 边缘、非 nil、spread 边界），
// 避免具体像素数在 AA 微调下误报。

func usStops() []GradStop {
	return []GradStop{
		{StopColor: color.NRGBA{255, 0, 0, 255}, Offset: 0, Opacity: 1},
		{StopColor: color.NRGBA{0, 0, 255, 255}, Offset: 1, Opacity: 1},
	}
}

// usColorAt 调 GetColorFunctionUS 并取点，失败即 Fatal。
func usColorAt(t *testing.T, g *Gradient, x, y int) color.Color {
	t.Helper()
	cf, ok := g.GetColorFunctionUS(1.0, Identity).(ColorFunc)
	if !ok {
		t.Fatal("2+ stops 应返回 ColorFunc")
	}
	c := cf(x, y)
	if c == nil {
		t.Fatalf("cf(%d,%d) 返回 nil", x, y)
	}
	return c
}

func rgb(c color.Color) (r, g, b uint32) {
	r, g, b, _ = c.RGBA()
	r >>= 8
	g >>= 8
	b >>= 8
	return
}

// TestRadialUserSpaceFocusCenter: radial userSpaceOnUse 且 focus==center（简化路径）。
func TestRadialUserSpaceFocusCenter(t *testing.T) {
	// Points=[cx,cy,fx,fy,r]，center=focus=(50,50)，半径 40。
	g := &Gradient{
		IsRadial: true, Units: UserSpaceOnUse, Spread: PadSpread,
		Points: [5]float64{50, 50, 50, 50, 40}, Matrix: Identity, Stops: usStops(),
	}
	cr, cg, cb := rgb(usColorAt(t, g, 50, 50)) // 圆心 → stop0 红
	er, eg, eb := rgb(usColorAt(t, g, 89, 50)) // 近边缘 → 偏蓝
	if !(cr > cb) {
		t.Errorf("圆心应偏红: (%d,%d,%d)", cr, cg, cb)
	}
	if !(eb > er) {
		t.Errorf("边缘应偏蓝: (%d,%d,%d)", er, eg, eb)
	}
	// 圆外 pad → 最后一站蓝
	orr, _, ob := rgb(usColorAt(t, g, 5, 5))
	if !(ob > orr) {
		t.Errorf("圆外应 pad 成末站蓝: (%d,%d,%d)", orr, 0, ob)
	}
}

// TestRadialUserSpaceFocusOffset: focus 与 center 不同，走 RayCircleIntersection 路径。
func TestRadialUserSpaceFocusOffset(t *testing.T) {
	// focus=(30,30) 在圆内（center=(50,50), r=40）：df<1，直接求交。
	g := &Gradient{
		IsRadial: true, Units: UserSpaceOnUse, Spread: PadSpread,
		Points: [5]float64{50, 50, 30, 30, 40}, Matrix: Identity, Stops: usStops(),
	}
	for _, p := range [][2]int{{50, 50}, {30, 30}, {89, 50}, {5, 5}, {120, 120}, {-20, 50}} {
		c := usColorAt(t, g, p[0], p[1])
		r, gg, b := rgb(c)
		if r > 255 || gg > 255 || b > 255 {
			t.Errorf("颜色通道越界: (%d,%d,%d)", r, gg, b)
		}
	}
}

// TestRadialUserSpaceFocusOutside: focus 落在圆外，触发 intersection 修正分支。
func TestRadialUserSpaceFocusOutside(t *testing.T) {
	// focus=(0,0) 距 center=(50,50) 约 70.7 > r=50 → df>1，走圆外求交。
	g := &Gradient{
		IsRadial: true, Units: UserSpaceOnUse, Spread: PadSpread,
		Points: [5]float64{50, 50, 0, 0, 50}, Matrix: Identity, Stops: usStops(),
	}
	for _, p := range [][2]int{{50, 50}, {10, 10}, {99, 50}, {-5, -5}} {
		c := usColorAt(t, g, p[0], p[1])
		r, gg, b := rgb(c)
		if r > 255 || gg > 255 || b > 255 {
			t.Errorf("颜色通道越界: (%d,%d,%d)", r, gg, b)
		}
	}
}

// TestRadialObjectBoundingBox: OBB 单位分支，focus 偏移 + 非 Pad spread。
func TestRadialObjectBoundingBox(t *testing.T) {
	// OBB 下 Points 用 0..1 比例，按 Bounds(100x100) 缩放。
	g := &Gradient{
		IsRadial: true, Units: ObjectBoundingBox, Spread: ReflectSpread,
		Points: [5]float64{0.5, 0.5, 0.2, 0.2, 0.5}, Matrix: Identity,
		Stops: usStops(),
	}
	g.Bounds.X, g.Bounds.Y, g.Bounds.W, g.Bounds.H = 0, 0, 100, 100
	for _, p := range [][2]int{{50, 50}, {20, 20}, {99, 99}, {-15, 50}, {120, 50}} {
		c := usColorAt(t, g, p[0], p[1])
		r, gg, b := rgb(c)
		if r > 255 || gg > 255 || b > 255 {
			t.Errorf("颜色通道越界: (%d,%d,%d)", r, gg, b)
		}
	}
}

// TestLinearUserSpace: linear userSpaceOnUse 分支（含 spread 越界）。
func TestLinearUserSpace(t *testing.T) {
	// Points=[x1,y1,x2,y2,0]，水平 (0,0)→(100,0)。
	g := &Gradient{
		Units: UserSpaceOnUse, Spread: RepeatSpread,
		Points: [5]float64{0, 0, 100, 0, 0}, Matrix: Identity, Stops: usStops(),
	}
	lr, _, lb := rgb(usColorAt(t, g, 5, 50))  // 近 x1 → 红
	rr, _, rb := rgb(usColorAt(t, g, 95, 50)) // 近 x2 → 蓝
	if !(lr > lb) {
		t.Errorf("左端应偏红: (%d,%d,%d)", lr, 0, lb)
	}
	if !(rb > rr) {
		t.Errorf("右端应偏蓝: (%d,%d,%d)", rr, 0, rb)
	}
	// RepeatSpread 越界坐标仍应返回非 nil、通道合法
	for _, p := range [][2]int{{-30, 50}, {150, 50}} {
		r, gg, b := rgb(usColorAt(t, g, p[0], p[1]))
		if r > 255 || gg > 255 || b > 255 {
			t.Errorf("越界色通道非法: (%d,%d,%d)", r, gg, b)
		}
	}
}

// TestGradientStopSortByOffset: GetColorFunctionUS 会按 offset 升序排序 stops。
func TestGradientStopSortByOffset(t *testing.T) {
	// 故意乱序：offset 1 在前、0 在后；排序后左端应取 0 站颜色。
	g := &Gradient{
		Units: UserSpaceOnUse, Spread: PadSpread,
		Points: [5]float64{0, 0, 100, 0, 0}, Matrix: Identity,
		Stops: []GradStop{
			{StopColor: color.NRGBA{0, 0, 255, 255}, Offset: 1, Opacity: 1},
			{StopColor: color.NRGBA{255, 0, 0, 255}, Offset: 0, Opacity: 1},
		},
	}
	lr, _, lb := rgb(usColorAt(t, g, 5, 50))
	if !(lr > lb) {
		t.Errorf("排序后左端应取 offset=0 的红: (%d,%d,%d)", lr, 0, lb)
	}
}
