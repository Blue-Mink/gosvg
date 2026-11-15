package gosvg

import (
	"image"
	"strings"
	"testing"
)

// renderWithDiagnostics 解析并渲染，返回像素与诊断列表，便于同时断言
// 「诊断正确」与「渲染决策零变化」。
func renderWithDiagnostics(t *testing.T, svg string) (*image.NRGBA, []string) {
	t.Helper()
	doc, err := Parse([]byte(svg))
	if err != nil {
		t.Fatal(err)
	}
	img, err := doc.Render(8, 8)
	if err != nil {
		t.Fatal(err)
	}
	return img, doc.Diagnostics()
}

// 无效 fill / stroke 声明按 CSS 语义被忽略、回退继承值（渲染不变），
// 但此前零诊断，-strict 用户无从得知颜色被悄悄改写。本轮补只读诊断。
func TestInvalidPaintProducesDiagnostic(t *testing.T) {
	for _, tc := range []struct {
		name string
		attr string
		want string
	}{
		{"invalid-literal", `fill="notacolor"`, "invalid fill value ignored: notacolor"},
		{"missing-hash", `fill="000000"`, "invalid fill value ignored: 000000"},
		{"invalid-hex", `fill="#gggggg"`, "invalid fill value ignored: #gggggg"},
		{"invalid-stroke", `fill="none" stroke="12px" stroke-width="2"`, "invalid stroke value ignored: 12px"},
		{"trailing-space", `fill="  bogus  "`, "invalid fill value ignored: bogus"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := renderWithDiagnostics(t,
				`<svg viewBox="0 0 4 4"><rect width="4" height="4" `+tc.attr+`/></svg>`)
			joined := strings.Join(diags, "\n")
			if !strings.Contains(joined, tc.want) {
				t.Fatalf("missing diagnostic %q; got:\n%s", tc.want, joined)
			}
		})
	}
}

// 合法值（含 none / currentColor / url 引用 / 可被 clamp 的 rgb 函数）不得误报。
func TestValidPaintProducesNoDiagnostic(t *testing.T) {
	for _, v := range []string{
		"red", "#fff", "#ff000080", "rgb(1,2,3)", "hsl(120 50% 50%)",
		"none", "transparent", "currentColor", "rgb(999,0,0)",
	} {
		t.Run(v, func(t *testing.T) {
			_, diags := renderWithDiagnostics(t,
				`<svg viewBox="0 0 4 4"><rect width="4" height="4" fill="`+v+`"/></svg>`)
			joined := strings.Join(diags, "\n")
			if strings.Contains(joined, "invalid fill") || strings.Contains(joined, "invalid stroke") {
				t.Fatalf("valid paint %q should not be diagnosed; got:\n%s", v, joined)
			}
		})
	}
}

// 容器 <svg>/<g> 上的无效声明会经 ctx.style 影响整棵子树继承值，也应报诊断。
func TestInvalidPaintOnContainerIsDiagnosed(t *testing.T) {
	_, diags := renderWithDiagnostics(t,
		`<svg viewBox="0 0 4 4" fill="000000"><g stroke="bogus"><rect width="4" height="4"/></g></svg>`)
	joined := strings.Join(diags, "\n")
	for _, want := range []string{
		"invalid fill value ignored: 000000",
		"invalid stroke value ignored: bogus",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing diagnostic %q; got:\n%s", want, joined)
		}
	}
}

// 纯定义节点上的 fill / stroke 对渲染无影响，不得误报。
func TestInvalidPaintOnDefinitionNodeIsNotDiagnosed(t *testing.T) {
	for _, body := range []string{
		`<defs fill="000000"><linearGradient id="g"/></defs>`,
		`<mask id="m" fill="bogus"><rect width="4" height="4"/></mask>`,
		`<clipPath id="c" stroke="12px"><rect width="4" height="4"/></clipPath>`,
		`<marker id="mk" fill="bogus"/>`,
		`<pattern id="p" stroke="bogus" width="1" height="1"/>`,
		`<stop offset="0" fill="bogus"/>`,
	} {
		_, diags := renderWithDiagnostics(t,
			`<svg viewBox="0 0 4 4">`+body+`<rect width="4" height="4" fill="blue"/></svg>`)
		joined := strings.Join(diags, "\n")
		if strings.Contains(joined, "invalid fill") || strings.Contains(joined, "invalid stroke") {
			t.Errorf("definition node should not be diagnosed (%s); got:\n%s", body, joined)
		}
	}
}

// CSS / 内联 style 最终也写回 n.attrs，应被同一处检查覆盖。
func TestInvalidPaintInInlineStyleIsDiagnosed(t *testing.T) {
	_, diags := renderWithDiagnostics(t,
		`<svg viewBox="0 0 4 4"><rect width="4" height="4" style="fill:000000"/></svg>`)
	joined := strings.Join(diags, "\n")
	if !strings.Contains(joined, "invalid fill value ignored: 000000") {
		t.Fatalf("inline style diagnostic missing; got:\n%s", joined)
	}
}

// 诊断按文本去重；同一无效值出现多次只报一条。
func TestInvalidPaintDiagnosticsAreDeduplicated(t *testing.T) {
	_, diags := renderWithDiagnostics(t,
		`<svg viewBox="0 0 4 4"><rect width="2" height="4" fill="000000"/>`+
			`<rect x="2" width="2" height="4" fill="000000"/></svg>`)
	n := 0
	for _, d := range diags {
		if d == "invalid fill value ignored: 000000" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("diagnostic should be deduplicated, got %d entries: %v", n, diags)
	}
}

// 最关键的不变量：诊断是纯只读的，渲染像素与修复前逐字节一致。
// 与 colorfallback_test.go 的既有断言呼应（无效声明回退 black、none 不绘制）。
func TestInvalidPaintDiagnosticsDoNotChangePixels(t *testing.T) {
	img, _ := renderWithDiagnostics(t,
		`<svg viewBox="0 0 4 4"><rect width="4" height="4" fill="000000"/></svg>`)
	if got := img.NRGBAAt(2, 2); got.R != 0 || got.G != 0 || got.B != 0 || got.A != 255 {
		t.Fatalf("invalid fill should still fall back to opaque black, got %v", got)
	}
	img2, _ := renderWithDiagnostics(t,
		`<svg viewBox="0 0 4 4"><rect width="4" height="4" fill="none"/></svg>`)
	if got := img2.NRGBAAt(2, 2); got.A != 0 {
		t.Fatalf(`fill="none" should still mean no fill, got %v`, got)
	}
	img3, _ := renderWithDiagnostics(t,
		`<svg viewBox="0 0 4 4"><rect width="4" height="4" fill="red"/></svg>`)
	if got := img3.NRGBAAt(2, 2); got.R != 255 || got.A != 255 {
		t.Fatalf("valid red should still draw, got %v", got)
	}
}
