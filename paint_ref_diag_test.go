package gosvg

import (
	"strings"
	"testing"
)

// 本文件锁定 fill/stroke 的 url(#id) 引用诊断。
//
// 修复前：resolveGradient 在 id 不是 gradient 时无条件报
// "gradient reference not found: <id>"，但同样的 url(#id) 完全可能是合法的
// <pattern> 引用 —— 结果合法 pattern 被误报，-strict 会因此错误退出 3。
// 修复后：诊断移到 renderShape 分派链末端，只有 gradient 与 pattern 都未命中
// 才报中性的 "paint reference not resolved: <id>"。纯只读，像素零变化。

const patternDiagSVG = `<defs><pattern id="p" width="2" height="2" patternUnits="userSpaceOnUse">` +
	`<rect width="2" height="2" fill="blue"/></pattern></defs>`

const gradientDiagSVG = `<defs><linearGradient id="g"><stop offset="0" stop-color="red"/>` +
	`</linearGradient></defs>`

// 核心回归：合法的 <pattern> / <gradient> 引用必须零诊断，且像素正确。
func TestValidPaintReferenceHasNoDiagnostic(t *testing.T) {
	for _, tc := range []struct {
		name  string
		defs  string
		attr  string
		wantR uint8
		wantB uint8
	}{
		{"pattern-fill", patternDiagSVG, `fill="url(#p)"`, 0, 255},
		{"gradient-fill", gradientDiagSVG, `fill="url(#g)"`, 255, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			img, diags := renderWithDiagnostics(t,
				`<svg viewBox="0 0 4 4">`+tc.defs+
					`<rect width="4" height="4" `+tc.attr+`/></svg>`)
			joined := strings.Join(diags, "\n")
			if strings.Contains(joined, "reference not found") ||
				strings.Contains(joined, "not resolved") {
				t.Fatalf("valid %s must not be diagnosed; got:\n%s", tc.name, joined)
			}
			px := img.NRGBAAt(2, 2)
			if px.R != tc.wantR || px.B != tc.wantB || px.A != 255 {
				t.Fatalf("%s should still draw correctly, got %v", tc.name, px)
			}
		})
	}
}

// url(#id) 解析不了时，报一条中性诊断（id 不存在 / 指向非 paint 元素）。
func TestUnresolvedPaintReferenceIsDiagnosed(t *testing.T) {
	for _, tc := range []struct {
		name string
		defs string
		id   string
	}{
		{"missing-id", "", "nope"},
		{"clipPath-id", `<defs><clipPath id="c"><rect width="1" height="1"/></clipPath></defs>`, "c"},
		{"mask-id", `<defs><mask id="m"><rect width="1" height="1" fill="white"/></mask></defs>`, "m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := renderWithDiagnostics(t,
				`<svg viewBox="0 0 4 4">`+tc.defs+
					`<rect width="4" height="4" fill="url(#`+tc.id+`)"/></svg>`)
			joined := strings.Join(diags, "\n")
			want := "paint reference not resolved: " + tc.id
			if !strings.Contains(joined, want) {
				t.Fatalf("missing diagnostic %q; got:\n%s", want, joined)
			}
			if strings.Contains(joined, "gradient reference not found") {
				t.Fatalf("old misleading wording must be gone; got:\n%s", joined)
			}
		})
	}
}

// 历史措辞 "gradient reference not found" 已彻底不再产生。
func TestGradientReferenceNotFoundWordingGone(t *testing.T) {
	// 全仓唯一曾产生该字符串的路径：id 不是 gradient。
	_, diags := renderWithDiagnostics(t,
		`<svg viewBox="0 0 4 4"><rect width="4" height="4" fill="url(#nope)"/></svg>`)
	for _, d := range diags {
		if strings.Contains(d, "gradient reference not found") {
			t.Fatalf("stale wording still emitted: %q", d)
		}
	}
}

// 合法不绘制 / 空描边 / 明确 stroke-width=0 不得误报。
func TestNoPaintReferenceDiagnosticForLegitimateNoDraw(t *testing.T) {
	for _, tc := range []struct {
		name string
		attr string
	}{
		{"fill-none", `fill="none"`},
		{"fill-transparent", `fill="transparent"`},
		{"default-no-stroke", `fill="red"`},
		{"missing-ref-zero-width", `fill="red" stroke="url(#nope)" stroke-width="0"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := renderWithDiagnostics(t,
				`<svg viewBox="0 0 4 4"><rect width="4" height="4" `+tc.attr+`/></svg>`)
			joined := strings.Join(diags, "\n")
			if strings.Contains(joined, "not resolved") ||
				strings.Contains(joined, "reference not found") {
				t.Fatalf("%s should not be diagnosed; got:\n%s", tc.name, joined)
			}
		})
	}
}

// 描边的 url(#不存在) 在真正要描边时也要报（fill 与 stroke 两处都覆盖）。
func TestUnresolvedStrokeReferenceIsDiagnosed(t *testing.T) {
	_, diags := renderWithDiagnostics(t,
		`<svg viewBox="0 0 4 4"><rect width="4" height="4" fill="red" `+
			`stroke="url(#nope)" stroke-width="2"/></svg>`)
	joined := strings.Join(diags, "\n")
	if !strings.Contains(joined, "paint reference not resolved: nope") {
		t.Fatalf("stroke diagnostic missing; got:\n%s", joined)
	}
}

// stroke="url(#pattern)" 仍按既有子集边界报「不支持」，不得被新诊断覆盖。
func TestStrokePatternUnsupportedDiagnosticUnchanged(t *testing.T) {
	_, diags := renderWithDiagnostics(t,
		`<svg viewBox="0 0 4 4">`+patternDiagSVG+
			`<rect width="4" height="4" fill="red" stroke="url(#p)" stroke-width="2"/></svg>`)
	joined := strings.Join(diags, "\n")
	if !strings.Contains(joined, "stroke with pattern reference is not supported") {
		t.Fatalf("existing diagnostic lost; got:\n%s", joined)
	}
	if strings.Contains(joined, "paint reference not resolved") {
		t.Fatalf("should not also report unresolved; got:\n%s", joined)
	}
}

// 诊断按文本去重：同一无效 id 出现多次只报一条。
func TestUnresolvedPaintReferenceIsDeduplicated(t *testing.T) {
	_, diags := renderWithDiagnostics(t,
		`<svg viewBox="0 0 4 4"><rect width="2" height="4" fill="url(#nope)"/>`+
			`<rect x="2" width="2" height="4" fill="url(#nope)"/></svg>`)
	n := 0
	for _, d := range diags {
		if d == "paint reference not resolved: nope" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("diagnostic should be deduplicated, got %d entries: %v", n, diags)
	}
}

// 最关键的不变量：诊断调整是纯只读的，像素逐字节不变。
func TestPaintReferenceDiagnosticsDoNotChangePixels(t *testing.T) {
	for _, tc := range []struct {
		name       string
		svg        string
		r, g, b, a uint8
	}{
		{"pattern", `<svg viewBox="0 0 4 4">` + patternDiagSVG +
			`<rect width="4" height="4" fill="url(#p)"/></svg>`, 0, 0, 255, 255},
		{"gradient", `<svg viewBox="0 0 4 4">` + gradientDiagSVG +
			`<rect width="4" height="4" fill="url(#g)"/></svg>`, 255, 0, 0, 255},
		{"missing", `<svg viewBox="0 0 4 4"><rect width="4" height="4" fill="url(#nope)"/></svg>`, 0, 0, 0, 0},
		{"plain", `<svg viewBox="0 0 4 4"><rect width="4" height="4" fill="red"/></svg>`, 255, 0, 0, 255},
	} {
		t.Run(tc.name, func(t *testing.T) {
			img, _ := renderWithDiagnostics(t, tc.svg)
			got := img.NRGBAAt(2, 2)
			if got.R != tc.r || got.G != tc.g || got.B != tc.b || got.A != tc.a {
				t.Fatalf("pixel changed: got %v want (%d,%d,%d,%d)",
					got, tc.r, tc.g, tc.b, tc.a)
			}
		})
	}
}
