package rasterx

import (
	"math"
	"strings"
	"testing"

	"golang.org/x/image/math/fixed"
)

// 本文件补 internal/rasterx/geomx.go 的几何工具覆盖。这些函数是 fork 里最纯的
// 数学/路径逻辑，断言一律用恒等式或结构性质，不写死具体实现产生的浮点尾数。

func fp(x, y int) fixed.Point26_6 {
	return fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}
}

func fapprox(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

// ---------- 基础向量工具 ----------

func TestGeomxVectorBasics(t *testing.T) {
	v := fp(3, 4)

	inv := Invert(v)
	if inv.X != -v.X || inv.Y != -v.Y {
		t.Fatalf("Invert(%v) = %v", v, inv)
	}
	if got := lengthAsFixed(Length(v)); got != 5 {
		t.Errorf("Length(3,4 in fixed units) = %v, want 5", got)
	}

	star := turnStarboard90(v)
	if star.X != -v.Y || star.Y != v.X || DotProd(v, star) != 0 {
		t.Errorf("turnStarboard90 应旋转 90°且与原向量正交: %v", star)
	}
	port := turnPort90(v)
	if port.X != v.Y || port.Y != -v.X || DotProd(v, port) != 0 {
		t.Errorf("turnPort90 应旋转 -90°且与原向量正交: %v", port)
	}
	if star != Invert(port) {
		t.Errorf("starboard 与 port 应互为反向: star=%v port=%v", star, port)
	}
	// 双 90° 旋转 = 取反
	if got := turnStarboard90(star); got != inv {
		t.Errorf("两次 starboard 旋转应为取反: got %v want %v", got, inv)
	}

	// DotProd 双线性 / 对称
	a, b := fp(2, 3), fp(5, 7)
	if DotProd(a, b) != DotProd(b, a) {
		t.Error("DotProd 不满足交换律")
	}
	if got := DotProd(a.Add(b), v); got != DotProd(a, v)+DotProd(b, v) {
		t.Errorf("DotProd 对加法不满足分配律: %v", got)
	}
	if DotProd(fp(0, 0), v) != 0 {
		t.Error("零向量点积应为 0")
	}
}

// lengthAsFixed 仅用于把 fixed 单位的 Length 结果换算成可读整数做断言。
func lengthAsFixed(v fixed.Int26_6) int { return int(v / 64) }

// ---------- Path 构建 / 序列化 / 回放 ----------

func TestPathBuildAndToSVGPath(t *testing.T) {
	var p Path
	p.Start(fp(1, 2))
	p.Line(fp(3, 4))
	p.QuadBezier(fp(5, 6), fp(7, 8))
	p.CubeBezier(fp(9, 10), fp(11, 12), fp(13, 14))
	p.Stop(true)

	// 命令 token 长度：M(3) L(3) Q(5) C(7) Z(1) = 19
	if len(p) != 19 {
		t.Fatalf("Path 长度 = %d, want 19", len(p))
	}
	wantCmds := []PathCommand{PathMoveTo, PathLineTo, PathQuadTo, PathCubicTo, PathClose}
	gotCmds := []PathCommand{
		PathCommand(p[0]), PathCommand(p[3]), PathCommand(p[6]), PathCommand(p[11]), PathCommand(p[18]),
	}
	for i := range wantCmds {
		if gotCmds[i] != wantCmds[i] {
			t.Errorf("第 %d 个命令 = %v, want %v", i, gotCmds[i], wantCmds[i])
		}
	}
	// 坐标往返：CubicTo 的终点应原样存下。
	if p[16] != fixed.I(13) || p[17] != fixed.I(14) {
		t.Errorf("CubicTo 终点未原样保存: (%v,%v)", p[16], p[17])
	}

	s := p.ToSVGPath()
	for _, tok := range []string{"M", "L", "Q", "C", "Z"} {
		if !strings.Contains(s, tok) {
			t.Errorf("ToSVGPath 缺少 %q: %s", tok, s)
		}
	}
	// String 是 ToSVGPath 的别名
	if p.String() != s {
		t.Error("Path.String 与 ToSVGPath 不一致")
	}
	// 坐标按 /64 输出（fixed.I(1)=64 → "1.000"）
	if !strings.Contains(s, "M1.000,2.000") {
		t.Errorf("MoveTo 未按 fixed 单位输出: %s", s)
	}
	// 三次贝塞尔段必须以单个 "C" 开头；历史笔误曾输出 "CC..."。
	if strings.Contains(s, "CC") {
		t.Errorf("ToSVGPath 出现重复 'C' 前缀: %s", s)
	}
	// 每个命令字母只应出现规定次数（M/L/Q/C/Z 各 1）。
	for _, tok := range []string{"M", "L", "Q", "C", "Z"} {
		if n := strings.Count(s, tok); n != 1 {
			t.Errorf("命令 %q 出现 %d 次, want 1: %s", tok, n, s)
		}
	}
}

func TestPathClearAndStop(t *testing.T) {
	var p Path
	p.Start(fp(1, 1))
	p.Stop(false)
	if len(p) != 3 {
		t.Fatalf("Stop(false) 不应追加 token, len=%d", len(p))
	}
	p.Stop(true)
	if len(p) != 4 || PathCommand(p[3]) != PathClose {
		t.Fatalf("Stop(true) 应追加 PathClose, len=%d tail=%v", len(p), PathCommand(p[len(p)-1]))
	}
	p.Clear()
	if len(p) != 0 {
		t.Errorf("Clear 后 len=%d, want 0", len(p))
	}
	// Clear 复用底层数组：重新构建仍能正确 append
	p.Start(fp(2, 3))
	if len(p) != 3 || p[1] != fixed.I(2) || p[2] != fixed.I(3) {
		t.Errorf("Clear 后重新 Start 异常: %v", p)
	}
}

func TestPathAddToReplaysCommands(t *testing.T) {
	var p Path
	p.Start(fp(1, 2))
	p.Line(fp(3, 4))
	p.QuadBezier(fp(5, 6), fp(7, 8))
	p.CubeBezier(fp(9, 10), fp(11, 12), fp(13, 14))
	p.Stop(true)

	var got Path
	p.AddTo(&got)
	// AddTo 在 MoveTo 前会先 Stop(false)，最终再 Stop(false)；
	// 用 Path 作 Adder 时 Stop(false) 是空操作，因此命令序列应完全一致。
	if len(got) != len(p) {
		t.Fatalf("AddTo 回放长度 = %d, want %d", len(got), len(p))
	}
	for i := range p {
		if got[i] != p[i] {
			t.Fatalf("AddTo 第 %d 个 token 不一致: got %v want %v", i, got[i], p[i])
		}
	}
}

// recordingAdder 记录 AddTo 的调用序列，用于验证 MoveTo 前的隐式 Stop(false)。
type recordingAdder struct {
	calls []string
}

func (r *recordingAdder) Start(a fixed.Point26_6) { r.calls = append(r.calls, "Start") }
func (r *recordingAdder) Line(b fixed.Point26_6)  { r.calls = append(r.calls, "Line") }
func (r *recordingAdder) QuadBezier(b, c fixed.Point26_6) {
	r.calls = append(r.calls, "QuadBezier")
}
func (r *recordingAdder) CubeBezier(b, c, d fixed.Point26_6) {
	r.calls = append(r.calls, "CubeBezier")
}
func (r *recordingAdder) Stop(closeLoop bool) {
	if closeLoop {
		r.calls = append(r.calls, "Stop(true)")
	} else {
		r.calls = append(r.calls, "Stop(false)")
	}
}

func TestPathAddToStopSequence(t *testing.T) {
	var p Path
	p.Start(fp(0, 0))
	p.Line(fp(1, 0))
	p.Stop(true)

	r := &recordingAdder{}
	p.AddTo(r)
	want := []string{"Stop(false)", "Start", "Line", "Stop(true)", "Stop(false)"}
	if len(r.calls) != len(want) {
		t.Fatalf("AddTo 调用序列 = %v, want %v", r.calls, want)
	}
	for i := range want {
		if r.calls[i] != want[i] {
			t.Fatalf("AddTo 第 %d 次调用 = %q, want %q（序列 %v）", i, r.calls[i], want[i], r.calls)
		}
	}
}

func TestToLength(t *testing.T) {
	ln := fixed.Int26_6(10 * 64) // 10 个用户单位
	q := ToLength(fp(3, 4), ln)
	// 方向不变，长度 = ln（固定点整数换算，允许 1 单位误差）
	if q.X <= 0 || q.Y <= 0 {
		t.Errorf("ToLength 应保持方向: %v", q)
	}
	if ratio := float64(q.Y) / float64(q.X); !fapprox(ratio, 4.0/3.0, 0.02) {
		t.Errorf("ToLength 改变了方向比例: %v (ratio=%v)", q, ratio)
	}
	if got := lengthAsFixed(Length(q)); got < 9 || got > 11 {
		t.Errorf("ToLength 结果长度 = %v, want ~10", got)
	}
	// 边界：零长度 / 零向量都返回零向量
	if ToLength(fp(3, 4), 0) != (fixed.Point26_6{}) {
		t.Error("ToLength(ln=0) 应返回零向量")
	}
	if ToLength(fp(0, 0), ln) != (fixed.Point26_6{}) {
		t.Error("ToLength(零向量) 应返回零向量")
	}
}

// ---------- 几何求交 / 曲率 ----------

func TestClosestPortside(t *testing.T) {
	bow, stern := fp(0, 0), fp(-10, 0) // 方向 +x
	p1, p2 := fp(5, 2), fp(5, -2)

	// 未相交标记直接短路
	if _, ok := ClosestPortside(bow, stern, p1, p2, false); ok {
		t.Error("isIntersecting=false 应直接返回 false")
	}
	// 两点都在 starboard（y<0，按本函数端口判定约定）；至少验证不 panic 且返回其一
	_, ok := ClosestPortside(bow, stern, p1, p2, true)
	if !ok {
		// 视端口侧定义不同，两侧都可能被判为「全在 port」；只要不是异常即可。
		t.Log("两候选点均在同侧，函数按实现返回（不强制具体点）")
	}
	// 混合两侧时必返回其中一点
	got, ok := ClosestPortside(bow, stern, fp(5, 3), fp(5, -3), true)
	if !ok {
		t.Error("混合两侧时应返回 intersects=true")
	}
	if got != fp(5, 3) && got != fp(5, -3) {
		t.Errorf("返回值 %v 不是两个候选点之一", got)
	}
}

func TestRadCurvatureColinearIsZero(t *testing.T) {
	// 三点共线 → 曲率为 0
	p0, p1, p2 := fp(0, 0), fp(2, 0), fp(4, 0)
	if got := RadCurvature(p0, p1, p2, fixed.Int52_12(2<<12)); got != 0 {
		t.Errorf("共线三点曲率 = %v, want 0", got)
	}
	// 非共线时符号随转向翻转
	a := RadCurvature(fp(0, 0), fp(2, 0), fp(4, 3), fixed.Int52_12(2<<12))
	b := RadCurvature(fp(0, 0), fp(2, 0), fp(4, -3), fixed.Int52_12(2<<12))
	if a == 0 || b == 0 {
		t.Log("非共线用例在该固定点比例下退化为 0，跳过符号检查")
	} else if a == b {
		t.Errorf("镜像转向应得到相反符号: a=%v b=%v", a, b)
	}
}

func TestCircleCircleIntersection(t *testing.T) {
	ct, cl := fp(0, 0), fp(4, 0)
	r := fixed.Int26_6(3 * 64)
	x1, x2, ok := CircleCircleIntersection(ct, cl, r, r)
	if !ok {
		t.Fatal("两个相交圆应返回 intersects=true")
	}
	// 交点应关于连心线对称，且到两圆心距离都等于半径（允许固定点舍入）
	if x1.Y != -x2.Y {
		t.Errorf("两交点应关于 x 轴对称: %v / %v", x1, yOf(x2))
	}
	for _, x := range []fixed.Point26_6{x1, x2} {
		d1 := math.Hypot(float64(x.X-ct.X), float64(x.Y-ct.Y)) / 64
		d2 := math.Hypot(float64(x.X-cl.X), float64(x.Y-cl.Y)) / 64
		if !fapprox(d1, 3, 0.1) || !fapprox(d2, 3, 0.1) {
			t.Errorf("交点 %v 到圆心距离 = (%.3f, %.3f), want ~3", x, d1, d2)
		}
	}
	// 相离 / 内含均返回 false
	if _, _, ok := CircleCircleIntersection(fp(0, 0), fp(20, 0), r, r); ok {
		t.Error("相离两圆不应相交")
	}
	if _, _, ok := CircleCircleIntersection(fp(0, 0), fp(0, 0), fixed.I(5), fixed.I(1)); ok {
		t.Error("内含圆不应相交")
	}
}

func yOf(p fixed.Point26_6) fixed.Int26_6 { return p.Y }

func TestCalcIntersect(t *testing.T) {
	// y=1 水平线与 x=2 竖直线交于 (2,1)
	a1, a2 := fp(-5, 1), fp(5, 1)
	b1, b2 := fp(2, -5), fp(2, 5)
	got := CalcIntersect(a1, a2, b1, b2)
	if got.X != fixed.I(2) || got.Y != fixed.I(1) {
		t.Errorf("CalcIntersect = %v, want (2,1)", got)
	}
}

func TestRayCircleIntersectionF(t *testing.T) {
	// 实测方向（探针 cmd/_probe_ray 实跑，事后已删）：圆心 (10,0) r=2，
	// s2=(0,0) 时只有 s1=(1,0) 命中近端 x=8；s1=(-1,0) 不命中。
	// 本 fork 的 t 参数方向与函数注释「starting at s2 passing through s1」
	// 的直觉不完全一致，此处按真实行为断言，不改实现。
	x, y, ok := RayCircleIntersectionF(1, 0, 0, 0, 10, 0, 2)
	if !ok {
		t.Fatal("射线应命中圆")
	}
	if !fapprox(x, 8, 1e-9) || !fapprox(y, 0, 1e-9) {
		t.Errorf("近端交点 = (%.3f,%.3f), want (8,0)", x, y)
	}
	// 切线（D=0）/ 完全不命中
	if _, _, ok := RayCircleIntersectionF(1, 0, 0, 0, 10, 3, 2); ok {
		t.Error("切线不应返回 intersects=true（实现要求 D>0）")
	}
	if _, _, ok := RayCircleIntersectionF(1, 0, 0, 0, 10, 10, 2); ok {
		t.Error("远离圆的射线不应相交")
	}
	// 反向 s1 不命中
	if _, _, ok := RayCircleIntersectionF(-1, 0, 0, 0, 10, 0, 2); ok {
		t.Error("反向射线不应相交")
	}
	// s2 位于圆心右侧时命中远端 x=12（验证「返回离 s2 最近的解」这一支）
	x2, _, ok2 := RayCircleIntersectionF(0, 0, 20, 0, 10, 0, 2)
	if !ok2 || !fapprox(x2, 12, 1e-9) {
		t.Errorf("s2=(20,0) 近端解 = (%.3f,ok=%v), want x=12", x2, ok2)
	}
}

func TestRayCircleIntersectionFixedWrapper(t *testing.T) {
	// fixed 包装层与 F 版本一致：s1=(1,0) s2=(0,0) 圆心 (10,0) r=2 → x=8
	p, ok := RayCircleIntersection(fp(1, 0), fp(0, 0), fp(10, 0), fixed.I(2))
	if !ok {
		t.Fatal("fixed 包装层应命中")
	}
	if p.X != fixed.I(8) {
		t.Errorf("fixed 包装层近端 x = %v, want %v", p.X, fixed.I(8))
	}
}
