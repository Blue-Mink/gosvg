package vector

// 本文件补 internal/vector 的白盒覆盖，重点不是「刷覆盖率」，而是守住
// 两类此前完全没被测到的语义：
//
//  1) fork 强制走 floating 路径（floatingPointMathThreshold 由上游 512 改为 0，
//     见 UPSTREAM.md），因此 fixedLineTo / fixedAccumulate* 在正常调用下是
//     「死路径」；amd64 又因 haveAccumulateSIMD=true 让标量 accumulate 全被
//     SSE4.1 汇编取代。这些标量 / 定点实现一旦被误改不会有任何现有测试报警，
//     故这里以同包白盒直接调用，锁定其数值语义不腐化（也方便非 amd64 回归）。
//  2) even-odd winding 是本 fork 相对 x/image/vector 的核心扩展；此前只有
//     evenodd_test.go 走公共 API 的 3 个冒烟测试，标量内核的「winding 计数 /
//     奇填偶镂 / 小数覆盖」从没被直接验证。这里用构造的扫描线 delta 序列
//     还原「外环填充、中心镂空」，并交叉比对 fixed 与 floating 两套实现。
//
// 断言一律用「0 / 非零 / 相对关系 / 饱和」而非具体像素值，避免 AA 相位与
// 浮点舍入误报；只有纯函数与定点换算才用精确值。

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"testing"
)

// ---------- 纯 helper / 定点换算 ----------

func TestPureHelpers(t *testing.T) {
	// lerp：端点和中点
	for _, tc := range []struct {
		t, px, py, qx, qy, wantX, wantY float32
	}{
		{0, 2, 4, 6, 8, 2, 4},
		{1, 2, 4, 6, 8, 6, 8},
		{0.5, 2, 4, 6, 8, 4, 6},
	} {
		x, y := lerp(tc.t, tc.px, tc.py, tc.qx, tc.qy)
		if x != tc.wantX || y != tc.wantY {
			t.Errorf("lerp(%v) = (%v,%v), want (%v,%v)", tc.t, x, y, tc.wantX, tc.wantY)
		}
	}

	// clamp：负 / 区间内 / 恰等宽 / 超界 四段
	for _, tc := range []struct {
		in, width int32
		want      uint
	}{
		{-1, 10, 0}, {0, 10, 0}, {5, 10, 5}, {10, 10, 10}, {11, 10, 10},
	} {
		if got := clamp(tc.in, tc.width); got != tc.want {
			t.Errorf("clamp(%d,%d) = %d, want %d", tc.in, tc.width, got, tc.want)
		}
	}

	// devSquared：三点共线为 0，中间点偏离则 >0，且与偏离方向无关
	if got := devSquared(0, 0, 5, 5, 10, 10); got != 0 {
		t.Errorf("devSquared(collinear) = %v, want 0", got)
	}
	if got := devSquared(0, 0, 5, 20, 10, 10); got <= 0 {
		t.Errorf("devSquared(curvy) = %v, want >0", got)
	}
	if a, b := devSquared(0, 0, 5, 20, 10, 10), devSquared(0, 0, 5, -10, 10, 10); a != b {
		t.Errorf("devSquared should be direction/sign agnostic: %v vs %v", a, b)
	}

	// fixed 定点换算（ϕ=9，fxOne=512）
	for _, tc := range []struct {
		x      int1ϕ
		fl, ce int32
	}{
		{fxOne, 1, 1},
		{fxOne - 1, 0, 1},
		{2*fxOne + 100, 2, 3},
		{0, 0, 0},
		{-1, -1, 0},
	} {
		if got := fixedFloor(tc.x); got != tc.fl {
			t.Errorf("fixedFloor(%d) = %d, want %d", tc.x, got, tc.fl)
		}
		if got := fixedCeil(tc.x); got != tc.ce {
			t.Errorf("fixedCeil(%d) = %d, want %d", tc.x, got, tc.ce)
		}
	}

	// floating 换算：正负、整数边界
	for _, tc := range []struct {
		x      float32
		fl, ce int32
	}{
		{1.5, 1, 2}, {1.0, 1, 1}, {0.0, 0, 0}, {-1.5, -2, -1}, {-0.2, -1, 0},
	} {
		if got := floatingFloor(tc.x); got != tc.fl {
			t.Errorf("floatingFloor(%v) = %d, want %d", tc.x, got, tc.fl)
		}
		if got := floatingCeil(tc.x); got != tc.ce {
			t.Errorf("floatingCeil(%v) = %d, want %d", tc.x, got, tc.ce)
		}
	}

	_ = math.Abs // 保持 math 依赖显式（曲线细分测试下方用到）
}

// ---------- fixed 标量 accumulate ----------
// 覆盖单位：int2ϕ，1.0 覆盖 = fxOne² = 1<<(2ϕ) = 262144。

const fxUnit = uint32(1 << (2 * ϕ)) // 262144
func negFx(n uint32) uint32         { return uint32(int32(-int32(n))) }

func TestFixedAccumulateScalars(t *testing.T) {
	// Mask：delta 序列 [0, +full, -full, 0] → 前缀和 0/full/0/0 → 中段饱和。
	buf := []uint32{0, fxUnit, negFx(fxUnit), 0}
	fixedAccumulateMask(buf)
	if buf[0] != 0 || buf[1] == 0 || buf[2] != 0 || buf[3] != 0 {
		t.Errorf("fixedAccumulateMask = %v, want [0,>0,0,0]", buf)
	}
	if buf[1] != 0xffff {
		t.Errorf("full coverage should saturate to 0xffff, got %#x", buf[1])
	}

	// OpSrc：同一 delta，满覆盖 → 0xff（不依赖 dst 初值）。
	dst := []uint8{0x11, 0x22, 0x33, 0x44}
	fixedAccumulateOpSrc(dst, []uint32{fxUnit, 0, 0, 0})
	for i, got := range dst {
		if got != 0xff {
			t.Errorf("fixedAccumulateOpSrc dst[%d] = %#x, want 0xff", i, got)
		}
	}

	// OpOver：满覆盖处忽略 dst 初值；mask 为 0 处保留 dst 初值。
	dst2 := []uint8{0x80, 0x40, 0x20, 0x10}
	fixedAccumulateOpOver(dst2, []uint32{0, 0, fxUnit, 0})
	if dst2[0] != 0x80 {
		t.Errorf("OpOver with zero mask must keep dst, got %#x", dst2[0])
	}
	if dst2[2] != 0xff || dst2[3] != 0xff {
		t.Errorf("OpOver after full mask should saturate, got %v", dst2)
	}

	// 三个标量 + evenodd 的 len guard：dst 比 src 短时不得 panic、不得写越界。
	guardSrc := []uint32{fxUnit, fxUnit}
	shortDst := []uint8{0x7f}
	fixedAccumulateOpSrc(shortDst, guardSrc)
	fixedAccumulateOpOver(shortDst, guardSrc)
	if shortDst[0] != 0x7f {
		t.Errorf("len guard must leave short dst untouched, got %#x", shortDst[0])
	}
	shortU32 := []uint32{0xdeadbeef}
	floatingAccumulateMask(shortU32, []float32{1, 1})
	floatingAccumulateOpSrc([]uint8{1}, []float32{1, 1})
	floatingAccumulateOpOver([]uint8{1}, []float32{1, 1})
	floatingAccumulateMaskEvenOdd(shortU32, []float32{1, 1})
	if shortU32[0] != 0xdeadbeef {
		t.Errorf("floating len guard must leave short dst untouched, got %#x", shortU32[0])
	}
}

// TestFixedAccumulateMaskEvenOdd 用扫描线 delta 还原「两个同向矩形」：
// 依次进入外矩形、进入内矩形、离开内矩形、离开外矩形。
// acc 依次 = 1,1,2,2,1,1,0,0（单位 fxUnit）→ winding 奇填偶镂。
func TestFixedAccumulateMaskEvenOdd(t *testing.T) {
	buf := []uint32{fxUnit, 0, fxUnit, 0, negFx(fxUnit), 0, negFx(fxUnit), 0}
	fixedAccumulateMaskEvenOdd(buf)
	wantFilled := []bool{true, true, false, false, true, true, false, false}
	for i, want := range wantFilled {
		if got := buf[i] != 0; got != want {
			t.Errorf("even-odd px[%d] filled=%v, want %v (buf=%v)", i, got, want, buf)
		}
	}
	// 而 non-zero 版本对同一 delta：路径内部 acc=2 的区间（px[2..5]）依然
	// 填充——这正是两条规则的分歧点；末尾 px[6..7] 已离开闭合路径，acc 回到 0，
	// 两种规则都应透明，不能拿来当分歧证据。
	nz := []uint32{fxUnit, 0, fxUnit, 0, negFx(fxUnit), 0, negFx(fxUnit), 0}
	fixedAccumulateMask(nz)
	for i := 2; i <= 5; i++ {
		if nz[i] == 0 {
			t.Errorf("non-zero px[%d] (winding=2) should be filled (nz=%v)", i, nz)
		}
	}
	if nz[6] != 0 || nz[7] != 0 {
		t.Errorf("both rules must be transparent after the path closes (nz=%v)", nz)
	}
}

// ---------- floating 标量 accumulate ----------

func TestFloatingAccumulateScalars(t *testing.T) {
	// Mask：|acc| 饱和、取绝对值；给一段负向 delta 验证 a<0 分支。
	dst := make([]uint32, 5)
	src := []float32{0, 1, -2, 0.5, 0} // acc: 0,1,-1,-0.5,-0.5
	floatingAccumulateMask(dst, src)
	if dst[0] != 0 {
		t.Errorf("floatingAccumulateMask[0] = %d, want 0", dst[0])
	}
	if dst[1] == 0 || dst[2] == 0 || dst[3] == 0 || dst[4] == 0 {
		t.Errorf("non-zero winding must stay covered, got %v", dst)
	}
	// 满覆盖（|acc|>=1）应饱和到 almost65536≈0xffff 附近，且两处相等。
	if dst[1] != dst[2] {
		t.Errorf("±1 coverage must be symmetric: %d vs %d", dst[1], dst[2])
	}
	if dst[1] < 0xff00 {
		t.Errorf("full coverage should be near 0xffff, got %d", dst[1])
	}

	// OpSrc / OpOver 与 fixed 版同语义：满覆盖饱和，OpOver 保留零 mask 的 dst。
	ds := []uint8{9, 9, 9, 9}
	floatingAccumulateOpSrc(ds, []float32{1, 0, 0, 0})
	for i, v := range ds {
		if v != 0xff {
			t.Errorf("floatingAccumulateOpSrc[%d] = %d, want 0xff", i, v)
		}
	}
	do := []uint8{0x80, 0x40, 0x20, 0x10}
	floatingAccumulateOpOver(do, []float32{0, 0, 1, 0})
	if do[0] != 0x80 {
		t.Errorf("OpOver zero mask must keep dst, got %#x", do[0])
	}
	if do[2] != 0xff || do[3] != 0xff {
		t.Errorf("OpOver after full mask should saturate, got %v", do)
	}
}

func TestFloatingAccumulateMaskEvenOdd(t *testing.T) {
	dst := make([]uint32, 8)
	src := []float32{1, 0, 1, 0, -1, 0, -1, 0}
	floatingAccumulateMaskEvenOdd(dst, src)
	wantFilled := []bool{true, true, false, false, true, true, false, false}
	for i, want := range wantFilled {
		if got := dst[i] != 0; got != want {
			t.Errorf("even-odd px[%d] filled=%v, want %v (dst=%v)", i, got, want, dst)
		}
	}
	// fractional winding：winding 落在 (1,2) 之间时按 ceil 计为 2 → 镂空，
	// 但小数覆盖部分仍参与 f = a-(w-1)。用 1.5 直接验证「奇/偶」判定边界。
	d2 := make([]uint32, 3)
	floatingAccumulateMaskEvenOdd(d2, []float32{0.5, 1.0, 0.5}) // acc .5,1.5,2
	if d2[0] == 0 || d2[1] != 0 || d2[2] != 0 {
		t.Errorf("fractional even-odd = %v, want [>0,0,0]", d2)
	}
}

// fixed 与 floating 是同一算法的两种数值实现，even-odd 的填充模式必须一致。
func TestEvenOddFixedFloatingConsistency(t *testing.T) {
	fixed := []uint32{fxUnit, 0, fxUnit, 0, negFx(fxUnit), 0, negFx(fxUnit), 0}
	floating := []float32{1, 0, 1, 0, -1, 0, -1, 0}
	fixedAccumulateMaskEvenOdd(fixed)
	fdst := make([]uint32, len(floating))
	floatingAccumulateMaskEvenOdd(fdst, floating)
	for i := range fixed {
		if (fixed[i] != 0) != (fdst[i] != 0) {
			t.Fatalf("px[%d]: fixed filled=%v but floating filled=%v", i, fixed[i] != 0, fdst[i] != 0)
		}
	}
}

// ---------- 几何 / 公共 API ----------

func rectPath(z *Rasterizer, x0, y0, x1, y1 float32) {
	z.MoveTo(x0, y0)
	z.LineTo(x1, y0)
	z.LineTo(x1, y1)
	z.LineTo(x0, y1)
	z.ClosePath()
}

func TestRasterizerGeometryAPI(t *testing.T) {
	z := NewRasterizer(20, 12)
	if z.Size() != (image.Pt(20, 12)) {
		t.Fatalf("Size = %v", z.Size())
	}
	if z.Bounds() != image.Rect(0, 0, 20, 12) {
		t.Fatalf("Bounds = %v", z.Bounds())
	}
	if z.DrawOp != draw.Over {
		t.Fatal("Reset must default DrawOp to draw.Over")
	}

	z.MoveTo(3, 4)
	if x, y := z.Pen(); x != 3 || y != 4 {
		t.Fatalf("Pen after MoveTo = (%v,%v)", x, y)
	}

	// QuadTo / CubeTo：细分后端点必须精确落在 (cx,cy)/(dx,dy)，
	// 且曲线段确实贡献了覆盖（不是被 devSquared 判为直线直接跳过）。
	z.QuadTo(10, 1, 18, 3)
	if x, y := z.Pen(); x != 18 || y != 3 {
		t.Fatalf("Pen after QuadTo = (%v,%v), want (18,3)", x, y)
	}
	z.CubeTo(15, 20, 6, 20, 2, 10)
	if x, y := z.Pen(); x != 2 || y != 10 {
		t.Fatalf("Pen after CubeTo = (%v,%v), want (2,10)", x, y)
	}
	z.ClosePath()
	if x, y := z.Pen(); x != 3 || y != 4 {
		t.Fatalf("Pen after ClosePath = (%v,%v), want back to (3,4)", x, y)
	}
	// bufF32 已被曲线/线段写入非零。
	nonzero := 0
	for _, v := range z.bufF32 {
		if v != 0 {
			nonzero++
		}
	}
	if nonzero == 0 {
		t.Fatal("curved path produced no coverage deltas")
	}

	// 长斜线触发 floatingLineTo 的「x1i > x0i+1」多像素分支（此前仅 54.2%）。
	z.Reset(20, 20)
	z.MoveTo(1, 1)
	z.LineTo(19, 19)
	z.LineTo(19, 1)
	z.ClosePath()
	nonzero = 0
	for _, v := range z.bufF32 {
		if v != 0 {
			nonzero++
		}
	}
	if nonzero == 0 {
		t.Fatal("diagonal path produced no coverage deltas")
	}

	// fixedLineTo 是 fork 强制 floating 后的死路径，白盒切过去验证仍可用。
	zf := NewRasterizer(8, 8)
	zf.setUseFloatingPointMath(false)
	if zf.useFloatingPointMath {
		t.Fatal("setUseFloatingPointMath(false) did not switch to fixed buffers")
	}
	rectPath(zf, 1, 1, 7, 7)
	fxNonzero := 0
	for _, v := range zf.bufU32 {
		if v != 0 {
			fxNonzero++
		}
	}
	if fxNonzero == 0 {
		t.Fatal("fixedLineTo produced no coverage deltas")
	}
}

func TestSetUseFloatingPointMathBuffers(t *testing.T) {
	z := NewRasterizer(4, 4)
	z.bufF32[0] = 0.75
	// 再次 set(true) 走「容量足够、就地清零」分支。
	z.setUseFloatingPointMath(true)
	if len(z.bufF32) != 16 || z.bufF32[0] != 0 {
		t.Fatalf("bufF32 reuse did not clear: len=%d [0]=%v", len(z.bufF32), z.bufF32[0])
	}
	// 放大尺寸走「重新 make」分支。
	z.Reset(8, 8)
	if len(z.bufF32) != 64 {
		t.Fatalf("after grow bufF32 len=%d, want 64", len(z.bufF32))
	}
	// 切到 fixed 走 bufU32 分配/复用分支。
	z.setUseFloatingPointMath(false)
	if len(z.bufU32) != 64 {
		t.Fatalf("fixed bufU32 len=%d, want 64", len(z.bufU32))
	}
	z.setUseFloatingPointMath(false) // 二次复用清零
	for i, v := range z.bufU32 {
		if v != 0 {
			t.Fatalf("fixed bufU32 reuse did not clear at %d: %d", i, v)
		}
	}
}

// ---------- Draw 各分支 ----------

func fillRectZ(w, h int) *Rasterizer {
	z := NewRasterizer(w, h)
	rectPath(z, 2, 2, float32(w)-2, float32(h)-2)
	return z
}

func TestDrawFastPathsAlphaAndRGBA(t *testing.T) {
	opaque := image.NewUniform(color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	const W, H = 12, 12

	// *image.Alpha + opaque uniform + Over → rasterizeDstAlphaSrcOpaqueOpOver (SIMD 直转)
	a := image.NewAlpha(image.Rect(0, 0, W, H))
	fillRectZ(W, H).Draw(a, a.Bounds(), opaque, image.Point{})
	if a.AlphaAt(6, 6).A == 0 || a.AlphaAt(0, 0).A != 0 {
		t.Errorf("Alpha/Over: center must be covered, corner clear (got center=%d corner=%d)",
			a.AlphaAt(6, 6).A, a.AlphaAt(0, 0).A)
	}

	// 同上但 DrawOp=Src → rasterizeDstAlphaSrcOpaqueOpSrc
	as := image.NewAlpha(image.Rect(0, 0, W, H))
	zs := fillRectZ(W, H)
	zs.DrawOp = draw.Src
	zs.Draw(as, as.Bounds(), opaque, image.Point{})
	if as.AlphaAt(6, 6).A == 0 {
		t.Error("Alpha/Src: center must be covered")
	}

	// *image.RGBA + uniform + Over → rasterizeDstRGBASrcUniformOpOver
	r := image.NewRGBA(image.Rect(0, 0, W, H))
	fillRectZ(W, H).Draw(r, r.Bounds(), opaque, image.Point{})
	if r.RGBAAt(6, 6).A == 0 || r.RGBAAt(0, 0).A != 0 {
		t.Errorf("RGBA/Over: center must be covered, corner clear")
	}

	// *image.RGBA + uniform + Src → rasterizeDstRGBASrcUniformOpSrc
	rs := image.NewRGBA(image.Rect(0, 0, W, H))
	zr := fillRectZ(W, H)
	zr.DrawOp = draw.Src
	zr.Draw(rs, rs.Bounds(), opaque, image.Point{})
	if rs.RGBAAt(6, 6).A == 0 {
		t.Error("RGBA/Src: center must be covered")
	}

	// *image.NRGBA dst 不匹配 fast path → 通用 rasterizeOpOver / OpSrc
	n := image.NewNRGBA(image.Rect(0, 0, W, H))
	fillRectZ(W, H).Draw(n, n.Bounds(), opaque, image.Point{})
	if n.NRGBAAt(6, 6).A == 0 {
		t.Error("NRGBA generic/Over: center must be covered")
	}
	ns := image.NewNRGBA(image.Rect(0, 0, W, H))
	zn := fillRectZ(W, H)
	zn.DrawOp = draw.Src
	zn.Draw(ns, ns.Bounds(), opaque, image.Point{})
	if ns.NRGBAAt(6, 6).A == 0 {
		t.Error("NRGBA generic/Src: center must be covered")
	}
}

// fast path 的逐像素子分支：r 与 dst.Bounds() 不等（dst 大一圈，r 仍等于 z.Bounds()）。
func TestDrawAlphaOpaqueSubRectBranch(t *testing.T) {
	const W, H = 8, 8
	opaque := image.NewUniform(color.NRGBA{R: 255, A: 255})
	dst := image.NewAlpha(image.Rect(0, 0, 10, 10)) // 比 z 大
	r := image.Rect(0, 0, W, H)                     // == z.Bounds() != dst.Bounds()
	z := fillRectZ(W, H)
	z.Draw(dst, r, opaque, image.Point{})
	if dst.AlphaAt(4, 4).A == 0 {
		t.Error("sub-rect branch: center must be covered")
	}
	if dst.AlphaAt(9, 9).A != 0 {
		t.Error("sub-rect branch: pixels outside r must stay untouched")
	}
	// OpSrc 的同一子分支
	dst2 := image.NewAlpha(image.Rect(0, 0, 10, 10))
	z2 := fillRectZ(W, H)
	z2.DrawOp = draw.Src
	z2.Draw(dst2, r, opaque, image.Point{})
	if dst2.AlphaAt(4, 4).A == 0 {
		t.Error("sub-rect Src branch: center must be covered")
	}
}

// DrawOp=Src 与 Over 的关键区别：mask=0 区域，Src 清空 dst，Over 保留 dst。
func TestDrawOpSrcClearsOutsideMask(t *testing.T) {
	const W, H = 12, 12
	opaque := image.NewUniform(color.NRGBA{R: 0, G: 0, B: 255, A: 255})
	prefill := func(img *image.RGBA) {
		draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{R: 255, A: 255}), image.Point{}, draw.Src)
	}
	over := image.NewRGBA(image.Rect(0, 0, W, H))
	prefill(over)
	fillRectZ(W, H).Draw(over, over.Bounds(), opaque, image.Point{})

	src := image.NewRGBA(image.Rect(0, 0, W, H))
	prefill(src)
	zs := fillRectZ(W, H)
	zs.DrawOp = draw.Src
	zs.Draw(src, src.Bounds(), opaque, image.Point{})

	if over.RGBAAt(0, 0).A == 0 {
		t.Error("Over must keep pre-filled pixels outside the mask")
	}
	if src.RGBAAt(0, 0).A != 0 {
		t.Error("Src must clear pixels outside the mask")
	}
	if over.RGBAAt(6, 6).A == 0 || src.RGBAAt(6, 6).A == 0 {
		t.Error("both ops must cover the mask interior")
	}
}

// even-odd 必须跳过 *image.Uniform fast path（否则 SIMD 绕过 accumulateMask，
// 镂空会失效）。这是对 UPSTREAM.md 第 3 条改动的回归锁。
func TestDrawEvenOddSkipsUniformFastPath(t *testing.T) {
	const size = 20
	opaque := image.NewUniform(color.NRGBA{R: 255, A: 255})
	build := func(eo bool) *image.NRGBA {
		z := NewRasterizer(size, size)
		z.MoveTo(2, 2)
		z.LineTo(18, 2)
		z.LineTo(18, 18)
		z.LineTo(2, 18)
		z.ClosePath()
		z.MoveTo(7, 7)
		z.LineTo(13, 7)
		z.LineTo(13, 13)
		z.LineTo(7, 13)
		z.ClosePath()
		z.UseEvenOdd = eo
		img := image.NewNRGBA(image.Rect(0, 0, size, size))
		z.Draw(img, img.Bounds(), opaque, image.Point{})
		return img
	}
	eo, nz := build(true), build(false)
	if eo.NRGBAAt(10, 10).A != 0 {
		t.Error("even-odd center must stay a hole even through Draw fast-path dispatch")
	}
	if nz.NRGBAAt(10, 10).A == 0 {
		t.Error("non-zero center must be filled")
	}
	// 用 *image.RGBA 再走一次（确保 RGBA 的 uniform 分支同样被 even-odd 跳过）。
	zr := NewRasterizer(size, size)
	zr.MoveTo(2, 2)
	zr.LineTo(18, 2)
	zr.LineTo(18, 18)
	zr.LineTo(2, 18)
	zr.ClosePath()
	zr.MoveTo(7, 7)
	zr.LineTo(13, 7)
	zr.LineTo(13, 13)
	zr.LineTo(7, 13)
	zr.ClosePath()
	zr.UseEvenOdd = true
	rgb := image.NewRGBA(image.Rect(0, 0, size, size))
	zr.Draw(rgb, rgb.Bounds(), opaque, image.Point{})
	if rgb.RGBAAt(10, 10).A != 0 {
		t.Error("even-odd center must stay a hole on *image.RGBA dst too")
	}
}
