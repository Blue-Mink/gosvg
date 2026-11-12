package gosvg

import (
	"testing"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
)

// eoAlpha 渲染 SVG 并返回若干采样点的 alpha。
func eoAlpha(t *testing.T, svg string, size int, pts ...[2]int) []uint8 {
	t.Helper()
	img, err := Render([]byte(svg), size)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	out := make([]uint8, len(pts))
	for i, p := range pts {
		out[i] = img.NRGBAAt(p[0], p[1]).A
	}
	return out
}

// TestEOConvertNestedSameDirection 同向嵌套（洞）是最典型的修复对象：
// 转换后走 nonzero 仍须镂空内圈。
func TestEOConvertNestedSameDirection(t *testing.T) {
	p := parsePathData(`M0 0h100v100H0zM20 20h60v60H20z`)
	converted, ok := convertEvenOddToNonZero(p)
	if !ok {
		t.Fatal("同向嵌套双 Z 子路径应可转换")
	}
	// 方向必须已交替（有向面积符号相反）。
	subs := splitEOSubpaths(converted)
	if len(subs) != 2 {
		t.Fatalf("转换后应有 2 条子路径，实际 %d", len(subs))
	}
	a0 := eoSignedArea(eoSamplePoly(subs[0]))
	a1 := eoSignedArea(eoSamplePoly(subs[1]))
	if (a0 > 0) == (a1 > 0) {
		t.Fatalf("转换后子路径方向未交替：%v / %v", a0, a1)
	}

	svg := `<svg viewBox="0 0 100 100"><path fill="#f00" fill-rule="evenodd" d="M0 0h100v100H0zM20 20h60v60H20z"/></svg>`
	got := eoAlpha(t, svg, 100, [2]int{50, 50}, [2]int{10, 50})
	if got[0] != 0 {
		t.Errorf("evenodd 内圈应镂空，alpha=%d", got[0])
	}
	if got[1] == 0 {
		t.Error("evenodd 外圈应填充")
	}
}

// TestEOConvertTripleWinding 三层同向嵌套：winding=3 填充、winding=2 镂空。
func TestEOConvertTripleWinding(t *testing.T) {
	svg := `<svg viewBox="0 0 100 100"><path fill="#f00" fill-rule="evenodd" d="M0 0h100v100H0zM20 20h60v60H20zM40 40h20v20H40z"/></svg>`
	got := eoAlpha(t, svg, 100,
		[2]int{50, 50}, [2]int{30, 50}, [2]int{10, 50})
	if got[0] == 0 {
		t.Error("winding=3（奇数）应填充")
	}
	if got[1] != 0 {
		t.Errorf("winding=2（偶数）应镂空，alpha=%d", got[1])
	}
	if got[2] == 0 {
		t.Error("winding=1（奇数）应填充")
	}
}

// TestEOConvertIndependent 两条完全独立、方向相同的子路径：都应填充。
func TestEOConvertIndependent(t *testing.T) {
	svg := `<svg viewBox="0 0 100 100"><path fill="#f00" fill-rule="evenodd" d="M0 0h40v40H0zM60 0h40v40H60z"/></svg>`
	got := eoAlpha(t, svg, 100, [2]int{20, 20}, [2]int{80, 20}, [2]int{50, 80})
	if got[0] == 0 || got[1] == 0 {
		t.Errorf("两个独立子路径都应填充，alpha=%d/%d", got[0], got[1])
	}
	if got[2] != 0 {
		t.Errorf("子路径外应为空，alpha=%d", got[2])
	}
}

// TestEOConvertCubicReverse 曲线子路径反转时控制点必须倒序，否则形状会变形。
// 用两段三次贝塞尔近似同向的两个圆角矩形轮廓。
func TestEOConvertCubicReverse(t *testing.T) {
	const outer = `M10 0 C60 0 100 40 100 50 C100 90 60 100 50 100 C20 100 0 80 0 50 C0 20 20 0 10 0 Z`
	const inner = `M30 30 C60 30 70 40 70 50 C70 70 60 70 50 70 C40 70 30 60 30 50 C30 40 30 30 30 30 Z`
	p := parsePathData(outer + inner)
	converted, ok := convertEvenOddToNonZero(p)
	if !ok {
		t.Fatal("同向曲线嵌套双 Z 子路径应可转换")
	}
	subs := splitEOSubpaths(converted)
	if len(subs) != 2 {
		t.Fatalf("转换后应有 2 条子路径，实际 %d", len(subs))
	}
	a0 := eoSignedArea(eoSamplePoly(subs[0]))
	a1 := eoSignedArea(eoSamplePoly(subs[1]))
	if (a0 > 0) == (a1 > 0) {
		t.Fatalf("曲线子路径方向未交替：%v / %v", a0, a1)
	}
	svg := `<svg viewBox="0 0 100 100"><path fill="#f00" fill-rule="evenodd" d="` + outer + inner + `"/></svg>`
	got := eoAlpha(t, svg, 100, [2]int{50, 50}, [2]int{5, 50})
	if got[0] != 0 {
		t.Errorf("曲线内圈应镂空，alpha=%d", got[0])
	}
	if got[1] == 0 {
		t.Error("曲线外圈应填充")
	}
}

// 自交的「8 字形」单条闭合轮廓 + 一条独立轮廓：等价转换不适用，必须拒绝。
func TestEOConvertRejectsSelfIntersection(t *testing.T) {
	const bowtie = `M10 10 L90 90 L90 10 L10 90 Z`
	const other = `M20 20 h10 v10 h-10 Z`
	p := parsePathData(bowtie + other)
	if _, ok := convertEvenOddToNonZero(p); ok {
		t.Error("自交子路径必须放弃转换，回退到原 even-odd 光栅化")
	}
	if !eoSelfIntersect(eoSamplePoly(splitEOSubpaths(p)[0])) {
		t.Error("自交检测应识别 8 字形轮廓")
	}
}

// 两条部分重叠（边界互相穿越）的子路径：非严格嵌套，转换不等价，必须拒绝。
func TestEOConvertRejectsPartialOverlap(t *testing.T) {
	const a = `M0 0 h60 v60 h-60 Z`
	const b = `M40 20 h60 v60 h-60 Z`
	p := parsePathData(a + b)
	if _, ok := convertEvenOddToNonZero(p); ok {
		t.Error("部分重叠子路径必须放弃转换")
	}
	polys := [][]eoPt{
		eoSamplePoly(splitEOSubpaths(p)[0]),
		eoSamplePoly(splitEOSubpaths(p)[1]),
	}
	if !eoPolysCross(polys[0], polys[1]) {
		t.Error("部分重叠检测应识别边界穿越")
	}
}

func TestEOPathEligible(t *testing.T) {
	d2 := `M0 0h100v100H0zM20 20h60v60H20z`
	p2 := parsePathData(d2)
	if !evenOddPathEligible(d2, p2) {
		t.Error("双 Z 子路径应可转换")
	}
	if !evenOddPathEligible(`M0 0 C1 1 2 2 3 3 Z M5 5 Q6 6 7 7 Z`, parsePathData(`M0 0 C1 1 2 2 3 3 Z M5 5 Q6 6 7 7 Z`)) {
		t.Error("C/Q 曲线子路径应可转换")
	}
	// 修正 10：A/a/T/t 由 parsePathData 归一化为 CubicTo/QuadTo，
	// 归一化后的段都是 eoSamplePoly 支持的，不再按原始 d 串拒绝。
	for _, d := range []string{
		`M0 0 A5 5 0 0 1 10 10 Z M20 20 h5 v5 h-5 Z`, // 弧线
		`M0 0 Q1 1 2 2 T3 3 4 4 Z M5 5 h2 v2 h-2 Z`,  // 平滑二次
	} {
		if !evenOddPathEligible(d, parsePathData(d)) {
			t.Errorf("A/T 归一化后应可转换：%s", d)
		}
	}
	// 单子路径仍不走「方向交替」转换（改由 renderShape 直接用 nonzero）。
	if evenOddPathEligible(`M0 0h100v100H0z`, parsePathData(`M0 0h100v100H0z`)) {
		t.Error("单子路径不应进入方向交替转换")
	}
}

// TestEOConvertSkippedInClipPath clipPath 内的 shape 由 clip-rule 映射进
// fillRule，语义属于裁剪光栅化，不应套用填充的等价转换。
func TestEOConvertSkippedInClipPath(t *testing.T) {
	inner := `<path clip-rule="evenodd" d="M0 0h10v10H0z M3 3h4v4H3z"/>`
	center, corner := renderClip(t, inner)
	if corner == 0 {
		t.Error("clip-rule=evenodd 外圈应保留")
	}
	if center != 0 {
		t.Errorf("clip-rule=evenodd 重叠区应排除，center alpha=%d", center)
	}
}

// TestEOConvertGradient 渐变填充也走转换后的 path。
func TestEOConvertGradient(t *testing.T) {
	svg := `<svg viewBox="0 0 100 100"><defs><linearGradient id="g"><stop offset="0" stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff"/></linearGradient></defs><path fill="url(#g)" fill-rule="evenodd" d="M0 0h100v100H0zM20 20h60v60H20z"/></svg>`
	got := eoAlpha(t, svg, 100, [2]int{50, 50}, [2]int{10, 50})
	if got[0] != 0 {
		t.Errorf("gradient evenodd 内圈应镂空，alpha=%d", got[0])
	}
	if got[1] == 0 {
		t.Error("gradient evenodd 外圈应填充")
	}
}

// 保证转换结果的 Path 结构合法（命令字 + 参数长度可被 AddTo 完整消费）。
func TestEOConvertedPathWellFormed(t *testing.T) {
	for _, d := range []string{
		`M0 0h100v100H0zM20 20h60v60H20z`,
		`M10 0 C60 0 100 40 100 50 C100 90 60 100 50 100 C20 100 0 80 0 50 C0 20 20 0 10 0 Z M30 30 C60 30 70 40 70 50 C70 70 60 70 50 70 C40 70 30 60 30 50 C30 40 30 30 30 30 Z`,
		`M0 0h100v100H0zM20 20h60v60H20zM40 40h20v20H40z`,
	} {
		out, ok := convertEvenOddToNonZero(parsePathData(d))
		if !ok {
			t.Fatalf("应可转换：%s", d)
		}
		for i := 0; i < len(out); {
			switch rasterx.PathCommand(out[i]) {
			case rasterx.PathMoveTo, rasterx.PathLineTo:
				i += 3
			case rasterx.PathQuadTo:
				i += 5
			case rasterx.PathCubicTo:
				i += 7
			case rasterx.PathClose:
				i++
			default:
				t.Fatalf("非法命令字 %d at %d (d=%s)", out[i], i, d)
			}
		}
	}
}
