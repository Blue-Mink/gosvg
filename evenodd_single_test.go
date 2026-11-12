package gosvg

import "testing"

// 修正 10 的渲染级回归：单子路径 + fill-rule="evenodd" 时，光栅器的
// even-odd 累积路径在非 1 缩放（viewBox 768→size 256）下会整块少画。
// 语义上单子路径 evenodd 与 nonzero 等价，故 renderShape 直接用 nonzero。
//
// 这是有意的「基线会 FAIL」测试：在修正 10 之前，(49,56) 等点 alpha=0。
// 只断言 helper 返回值不足以防止退化，必须验证真实渲染字节。
func TestEOSinglePathEvenOddFillsCompletely(t *testing.T) {
	const svg = `<svg viewBox="0 0 768 256"><path fill="black" fill-rule="evenodd" d="M0 0h768v256H341.3z"/></svg>`
	// 补齐区内的三个点：修正前 alpha=0，修正后不透明。
	pts := [][2]int{{49, 56}, {68, 57}, {89, 58}}
	got := eoAlpha(t, svg, 100, pts...)
	for i, a := range got {
		if a == 0 {
			t.Errorf("单子路径 evenodd 应在 (%d,%d) 填充，alpha=0（even-odd 累积路径少画）",
				pts[i][0], pts[i][1])
		}
	}
	// 外部点必须仍为透明，防止「改 nonzero 后画太满」的过度放宽。
	out := eoAlpha(t, svg, 100, [2]int{12, 57})
	if out[0] != 0 {
		t.Errorf("多边形外部点 (12,57) 应保持透明，alpha=%d", out[0])
	}
}

// eoSingleNonSelfIntersecting 的判定边界：自交单子路径必须拒绝，
// 否则 even-odd 会被误换成 nonzero，镂空语义丢失。
func TestEOSingleNonSelfIntersecting(t *testing.T) {
	cases := []struct {
		name string
		d    string
		want bool
	}{
		{"矩形单子路径", `M0 0h100v100H0z`, true},
		{"含小数坐标单子路径", `M0 0h768v256H341.3z`, true},
		{"含三次曲线单子路径", `M10 0 C60 0 100 40 100 50 C100 90 60 100 50 100 Z`, true},
		{"自交单子路径（五角星式）", `M10 10 L90 90 L90 10 L10 90 Z`, false},
		{"双 Z 子路径", `M0 0h100v100H0zM20 20h60v60H20z`, false},
		{"退化空路径", ``, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := eoSingleNonSelfIntersecting(parsePathData(c.d))
			if got != c.want {
				t.Errorf("eoSingleNonSelfIntersecting(%q) = %v, want %v", c.d, got, c.want)
			}
		})
	}
}

// 修正 10 放宽 A/a/T/t 门槛后，含弧线的多子路径走「方向交替 + nonzero」
// 转换，仍须正确镂空同向嵌套的内圈；不能直接把 fillRule 改成 nonzero。
func TestEOArcSubpathsStillHollowAfterConvert(t *testing.T) {
	const d = `M0 0 A50 50 0 0 1 100 0 L100 100 L0 100 Z M20 20 h60 v60 h-60 Z`
	if !evenOddPathEligible(d, parsePathData(d)) {
		t.Fatal("含弧线的多子路径应可转换")
	}
	svg := `<svg viewBox="0 0 100 100"><path fill="black" fill-rule="evenodd" d="` + d + `"/></svg>`
	got := eoAlpha(t, svg, 100, [2]int{50, 50}, [2]int{10, 50})
	if got[0] != 0 {
		t.Errorf("弧线 shape 内圈应镂空，alpha=%d", got[0])
	}
	if got[1] == 0 {
		t.Error("弧线 shape 外圈应填充")
	}
}

// clipPath 内的单子路径由 clip-rule 语义处理，不应被填充侧的 nonzero 分支影响。
// 单子路径本身无洞，evenodd 与 nonzero 都全填充；本测试固定这一行为，
// 防止有人把 renderShape 的分支误搬进 clipPath 路径。
func TestEOSinglePathClipRuleUnchanged(t *testing.T) {
	inner := `<path clip-rule="evenodd" d="M0 0h10v10H0z"/>`
	center, corner := renderClip(t, inner)
	if corner == 0 {
		t.Error("clip-rule=evenodd 外圈应保留")
	}
	if center == 0 {
		t.Error("单子路径无洞，clip 中心应填充（修正 10 前后一致）")
	}
}
