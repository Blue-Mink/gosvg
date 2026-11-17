package rasterx

import (
	"image"
	"image/color"
	"math"
	"testing"

	"golang.org/x/image/math/fixed"
)

// 本文件补 internal/rasterx 的冒烟覆盖。该 fork（~677 行）此前只有
// matrix_test.go 的性质测试，覆盖率约 3.9%；Filler / Gradient / Dasher /
// ScannerGV 这些被 850 语料每天调用的路径几乎裸奔。
// 断言用「非空 / 相对差异 / 边界」而非具体像素数，避免 AA 微调误报。

func countNRGBAOpaque(img *image.NRGBA) int {
	n := 0
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0 {
			n++
		}
	}
	return n
}

func nrgbaDiff(a, b *image.NRGBA) int {
	if a.Bounds() != b.Bounds() || len(a.Pix) != len(b.Pix) {
		return -1
	}
	n := 0
	for i := 0; i < len(a.Pix); i += 4 {
		if a.Pix[i] != b.Pix[i] || a.Pix[i+1] != b.Pix[i+1] ||
			a.Pix[i+2] != b.Pix[i+2] || a.Pix[i+3] != b.Pix[i+3] {
			n++
		}
	}
	return n
}

func newTestImage(w, h int) *image.NRGBA { return image.NewNRGBA(image.Rect(0, 0, w, h)) }

// ---------- Gradient / ApplyOpacity ----------

func TestApplyOpacity(t *testing.T) {
	full := ApplyOpacity(color.NRGBA{10, 20, 30, 255}, 1.0)
	zero := ApplyOpacity(color.NRGBA{10, 20, 30, 255}, 0.0)
	half := ApplyOpacity(color.NRGBA{10, 20, 30, 255}, 0.5)
	if full.A != 255 || zero.A != 0 {
		t.Fatalf("alpha 边界: full=%d zero=%d", full.A, zero.A)
	}
	if math.Abs(float64(half.A)-127.5) > 1 {
		t.Fatalf("半透明 alpha=%d, want ~128", half.A)
	}
	// RGB 不应被 opacity 改动（ApplyOpacity 只改 A）
	if full.R != 10 || full.G != 20 || full.B != 30 {
		t.Fatalf("RGB 被意外改动: %+v", full)
	}
}

func TestApplyOpacityClampsOutOfRange(t *testing.T) {
	// SVG 输入可含越界 opacity（fill-opacity="2" / "-1"）。float64→uint8
	// 转换是按 256 取模截断而非 clamp，故必须先夹到 [0,1]，否则：
	//   1.5  -> alpha 126 (应 255),  -0.5 -> alpha 129 (应 0)
	opaque := color.NRGBA{10, 20, 30, 255}
	if got := ApplyOpacity(opaque, 1.5); got.A != 255 {
		t.Errorf("opacity=1.5 alpha=%d, want 255 (clamp to 1)", got.A)
	}
	if got := ApplyOpacity(opaque, -0.5); got.A != 0 {
		t.Errorf("opacity=-0.5 alpha=%d, want 0 (clamp to 0)", got.A)
	}
	if got := ApplyOpacity(opaque, 2.0); got.A != 255 {
		t.Errorf("opacity=2.0 alpha=%d, want 255", got.A)
	}
	if got := ApplyOpacity(opaque, -1.0); got.A != 0 {
		t.Errorf("opacity=-1.0 alpha=%d, want 0", got.A)
	}
}

func TestGradientColorFuncStopCounts(t *testing.T) {
	// 0 / 1 个 stop 返回的是 color.Color（错误色 / 单色），2+ 才是 ColorFunc。
	for _, tc := range []struct {
		name  string
		stops []GradStop
	}{
		{"no stops", nil},
		{"one stop", []GradStop{{StopColor: color.NRGBA{1, 2, 3, 255}, Offset: 0, Opacity: 1}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := &Gradient{Stops: tc.stops, Spread: PadSpread}
			if _, ok := g.GetColorFunction(1.0).(color.Color); !ok {
				t.Fatal("0/1 stop 应返回 color.Color，而非 ColorFunc")
			}
		})
	}
	g := &Gradient{
		Stops: []GradStop{
			{StopColor: color.NRGBA{255, 0, 0, 255}, Offset: 0, Opacity: 1},
			{StopColor: color.NRGBA{0, 0, 255, 255}, Offset: 1, Opacity: 1},
		},
		Spread: PadSpread,
		// 水平线性渐变 (0,0)→(1,0)，objectBoundingBox 下按 Bounds 缩放。
		// 构造陷阱（非实现缺陷）：Points 全零 → d=0、t=NaN、两端取到同一 stop；
		// Matrix 留零值 → gradT 退化、Transform 恒返回 0，同样两端同色。
		// 根包 gradient.go:99 构造时显式传了 Matrix，故运行期不受影响。
		Points: [5]float64{0, 0, 1, 0, 0},
		Units:  ObjectBoundingBox,
		Matrix: Identity,
		Bounds: struct{ X, Y, W, H float64 }{0, 0, 100, 100},
	}
	cf, ok := g.GetColorFunction(1.0).(ColorFunc)
	if !ok {
		t.Fatal("2+ stops 应返回 ColorFunc")
	}
	a := cf(0, 0)
	b := cf(99, 0)
	if a == nil || b == nil {
		t.Fatal("ColorFunc 返回 nil")
	}
	// 线性渐变两端颜色应不同（否则 Points/Matrix 变换失效）
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	if ar == br && ag == bg && ab == bb {
		t.Error("线性渐变两端应取到不同 stop 颜色")
	}
}

func TestGradientSpreadMethodsDoNotPanic(t *testing.T) {
	stops := []GradStop{
		{StopColor: color.NRGBA{0, 0, 0, 255}, Offset: 0, Opacity: 1},
		{StopColor: color.NRGBA{255, 255, 255, 255}, Offset: 1, Opacity: 1},
	}
	for _, spread := range []SpreadMethod{PadSpread, ReflectSpread, RepeatSpread} {
		g := &Gradient{Stops: append([]GradStop(nil), stops...), Spread: spread,
			Bounds: struct{ X, Y, W, H float64 }{0, 0, 10, 10}}
		cf, ok := g.GetColorFunction(1.0).(ColorFunc)
		if !ok {
			t.Fatalf("spread=%d 未返回 ColorFunc", spread)
		}
		// 扫描一段含负/超界坐标，tColor 的 math.Mod / modRange 分支都要能跑
		for x := -15; x <= 25; x++ {
			if c := cf(x, x); c == nil {
				t.Fatalf("spread=%d x=%d 返回 nil", spread, x)
			}
		}
	}
}

// ---------- Filler：真正光栅化到 NRGBA ----------

// fillSquare 用 Filler 画一个 10..50 的正方形并 Draw，返回像素数。
func fillSquare(t *testing.T, evenOdd bool, closeLoop bool) (int, fixed.Rectangle26_6) {
	t.Helper()
	const W, H = 64, 64
	img := newTestImage(W, H)
	sc := NewScannerGV(W, H, img, img.Bounds())
	sc.SetColor(color.NRGBA{0, 0, 0, 255})
	sc.SetWinding(!evenOdd) // SetWinding(false) = even-odd
	f := NewFiller(W, H, sc)
	p := func(x, y int) fixed.Point26_6 {
		return fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}
	}
	f.Start(p(10, 10))
	f.Line(p(50, 10))
	f.Line(p(50, 50))
	f.Line(p(10, 50))
	f.Stop(closeLoop)
	sc.Draw()
	return countNRGBAOpaque(img), sc.GetPathExtent()
}

func TestFillerRendersSquare(t *testing.T) {
	for _, tc := range []struct {
		name      string
		evenOdd   bool
		closeLoop bool
	}{
		{"nonzero closed", false, true},
		{"evenodd closed", true, true},
		{"nonzero open (implicit close)", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, ext := fillSquare(t, tc.evenOdd, tc.closeLoop)
			if n == 0 {
				t.Fatal("Filler 未画出任何像素")
			}
			if ext.Min.X > fixed.I(10) || ext.Max.X < fixed.I(50) {
				t.Errorf("GetPathExtent 不含正方形范围: %+v", ext)
			}
		})
	}
}

func TestFillerClearResetsExtent(t *testing.T) {
	const W, H = 32, 32
	img := newTestImage(W, H)
	sc := NewScannerGV(W, H, img, img.Bounds())
	sc.SetColor(color.NRGBA{0, 0, 0, 255})
	f := NewFiller(W, H, sc)
	f.Start(fixed.Point26_6{X: fixed.I(5), Y: fixed.I(5)})
	f.Line(fixed.Point26_6{X: fixed.I(20), Y: fixed.I(20)})
	before := sc.GetPathExtent()
	f.Clear()
	after := sc.GetPathExtent()
	if before == after && (before.Min.X != 0 || before.Max.X != 0) {
		t.Errorf("Clear 后 extent 未重置: before=%+v after=%+v", before, after)
	}
	sc.Draw()
	if n := countNRGBAOpaque(img); n != 0 {
		t.Errorf("Clear 后仍画出 %d 像素", n)
	}
}

func TestFillerBezierCurvesRender(t *testing.T) {
	const W, H = 64, 64
	img := newTestImage(W, H)
	sc := NewScannerGV(W, H, img, img.Bounds())
	sc.SetColor(color.NRGBA{0, 0, 0, 255})
	f := NewFiller(W, H, sc)
	P := func(x, y int) fixed.Point26_6 {
		return fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}
	}
	f.Start(P(5, 32))
	f.QuadBezier(P(32, 5), P(59, 32)) // 二次贝塞尔
	f.Line(P(5, 32))
	f.Stop(true)
	sc.Draw()
	if n := countNRGBAOpaque(img); n == 0 {
		t.Fatal("QuadBezier 路径未画出任何像素")
	}

	img2 := newTestImage(W, H)
	sc2 := NewScannerGV(W, H, img2, img2.Bounds())
	sc2.SetColor(color.NRGBA{0, 0, 0, 255})
	f2 := NewFiller(W, H, sc2)
	f2.Start(P(5, 32))
	f2.CubeBezier(P(20, 5), P(44, 59), P(59, 32)) // 三次贝塞尔
	f2.Line(P(5, 32))
	f2.Stop(true)
	sc2.Draw()
	if n := countNRGBAOpaque(img2); n == 0 {
		t.Fatal("CubeBezier 路径未画出任何像素")
	}
	if d := nrgbaDiff(img, img2); d == 0 {
		t.Error("二次与三次贝塞尔应产生不同形状")
	}
}

func TestFillerDegenerateBezierFallsBackToLine(t *testing.T) {
	// 退化贝塞尔（r.a==b 等）应走 sgm.Line(d) 早退，不应 panic 或产生 NaN 像素。
	const W, H = 32, 32
	img := newTestImage(W, H)
	sc := NewScannerGV(W, H, img, img.Bounds())
	sc.SetColor(color.NRGBA{0, 0, 0, 255})
	f := NewFiller(W, H, sc)
	p := fixed.Point26_6{X: fixed.I(8), Y: fixed.I(8)}
	q := fixed.Point26_6{X: fixed.I(24), Y: fixed.I(24)}
	f.Start(p)
	f.QuadBezier(p, q)    // r.a==b
	f.CubeBezier(p, p, q) // 退化
	f.Stop(true)
	sc.Draw() // 只要不 panic 即可
	_ = countNRGBAOpaque(img)
}

// ---------- Dasher ----------

func drawDashedLine(t *testing.T, dashes []float64, offset float64) *image.NRGBA {
	t.Helper()
	const W, H = 64, 32
	img := newTestImage(W, H)
	sc := NewScannerGV(W, H, img, img.Bounds())
	sc.SetColor(color.NRGBA{0, 0, 0, 255})
	d := NewDasher(W, H, sc)
	d.SetStroke(fixed.I(3), fixed.I(4), ButtCap, nil, FlatGap, MiterClip, dashes, offset)
	d.Start(fixed.Point26_6{X: fixed.I(4), Y: fixed.I(16)})
	d.Line(fixed.Point26_6{X: fixed.I(60), Y: fixed.I(16)})
	d.Stop(false)
	sc.Draw()
	return img
}

func TestDasherDashArrayChangesOutput(t *testing.T) {
	solid := drawDashedLine(t, nil, 0)
	dashed := drawDashedLine(t, []float64{4, 4}, 0)
	ns, nd := countNRGBAOpaque(solid), countNRGBAOpaque(dashed)
	if ns == 0 {
		t.Fatal("实线未画出像素")
	}
	if nd == 0 {
		t.Fatal("虚线未画出像素")
	}
	if nd >= ns {
		t.Errorf("虚线像素应少于实线: dashed=%d solid=%d", nd, ns)
	}
	if d := nrgbaDiff(solid, dashed); d == 0 {
		t.Error("dash 数组未改变输出")
	}
}

func TestDasherAllZeroDashesFallsBackToSolid(t *testing.T) {
	solid := drawDashedLine(t, nil, 0)
	zero := drawDashedLine(t, []float64{0, -1, 0}, 0) // oneIsPos=false → 退化为实线
	if d := nrgbaDiff(solid, zero); d != 0 {
		t.Errorf("全零 dash 应退化为实线，diff=%d", d)
	}
}

func TestDasherDashOffsetShiftsPattern(t *testing.T) {
	a := drawDashedLine(t, []float64{6, 6}, 0)
	b := drawDashedLine(t, []float64{6, 6}, 3)
	if d := nrgbaDiff(a, b); d == 0 {
		t.Error("dashOffset 应平移虚线相位")
	}
}

// ---------- ScannerGV clip ----------

func TestScannerClipRestrictsDraw(t *testing.T) {
	const W, H = 64, 64
	img := newTestImage(W, H)
	sc := NewScannerGV(W, H, img, img.Bounds())
	sc.SetColor(color.NRGBA{0, 0, 0, 255})
	f := NewFiller(W, H, sc)
	P := func(x, y int) fixed.Point26_6 {
		return fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}
	}
	f.Start(P(0, 0))
	f.Line(P(64, 0))
	f.Line(P(64, 64))
	f.Line(P(0, 64))
	f.Stop(true)
	sc.SetClip(image.Rect(0, 0, 16, 16))
	sc.Draw()
	n := countNRGBAOpaque(img)
	if n == 0 {
		t.Fatal("裁剪区内应有像素")
	}
	if n > 16*16 {
		t.Errorf("裁剪未生效，像素数 %d 超出 16x16", n)
	}
}
