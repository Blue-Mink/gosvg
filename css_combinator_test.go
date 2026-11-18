package gosvg

import (
	"image"
	"testing"
)

// CSS 组合子（后代 / 子代 / 相邻兄弟 / 通用兄弟）测试。
//
// 通用布局：4 个水平排列的 10x10 rect（x=0/10/20/30，y=0），默认蓝色。
// viewBox 与输出同尺寸（40x10），1 单位 = 1px。给部分 rect 挂 class/id，
// CSS 规则命中时把 fill 改为红色，通过探测每个 rect 中心点（x=5+10*i, y=5）
// 是否为红色来断言匹配结果。这样不写死整图像素，只验证语义。

// renderCombo 渲染给定的 SVG（含 <style>），返回图像。
func renderCombo(t *testing.T, body, css string, w, h int) *image.NRGBA {
	t.Helper()
	data := []byte(`<svg width="` + itoa(w) + `" height="` + itoa(h) +
		`" viewBox="0 0 ` + itoa(w) + ` ` + itoa(h) + `"><style>` + css + `</style>` + body + `</svg>`)
	img, err := RenderTo(data, w, h)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return img
}

// itoa 是仅用于本文件的小整数转字符串（避免引入 strconv 依赖）。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// isRedAt 判断第 idx 个 10px 宽 rect 的中心像素是否为红色。
func isRedAt(img *image.NRGBA, idx int) bool {
	c := img.NRGBAAt(5+10*idx, 5)
	return c.A > 0 && c.R > 200 && c.G < 100 && c.B < 100
}

// TestCSSDescendantSelector 验证后代组合子 "A B"：B 是 A 的任意层级后代。
func TestCSSDescendantSelector(t *testing.T) {
	// <g class="outer"> 里有 <rect class="x">，<g> 外也有一个 <rect class="x">。
	// "g.outer rect.x" 只应命中 <g> 内的那个。
	body := `<rect class="x" x="0" y="0" width="10" height="10" fill="#0000ff"/>` +
		`<g class="outer"><rect class="x" x="10" y="0" width="10" height="10" fill="#0000ff"/></g>`
	img := renderCombo(t, body, `.outer .x{fill:#ff0000}`, 40, 10)
	if isRedAt(img, 0) {
		t.Error("g 外的 .x 不应被后代选择器命中")
	}
	if !isRedAt(img, 1) {
		t.Error("g.outer 内的 .x 应被后代选择器命中")
	}
}

// TestCSSDescendantMultiLevel 验证后代可跨越多层（.outer > g > rect）。
func TestCSSDescendantMultiLevel(t *testing.T) {
	body := `<g class="outer"><g><rect class="x" x="10" y="0" width="10" height="10" fill="#0000ff"/></g></g>`
	img := renderCombo(t, body, `.outer .x{fill:#ff0000}`, 40, 10)
	if !isRedAt(img, 1) {
		t.Error("多层后代应被 'A B' 命中（空格不限直接子级）")
	}
}

// TestCSSChildSelector 验证子代组合子 "A > B"：只命中直接子级。
func TestCSSChildSelector(t *testing.T) {
	// .outer 直接子 rect（idx1）应命中；嵌套在 <g> 里的 rect（idx2）不应命中。
	body := `<g class="outer">` +
		`<rect class="x" x="10" y="0" width="10" height="10" fill="#0000ff"/>` +
		`<g><rect class="x" x="20" y="0" width="10" height="10" fill="#0000ff"/></g>` +
		`</g>`
	img := renderCombo(t, body, `.outer > .x{fill:#ff0000}`, 40, 10)
	if !isRedAt(img, 1) {
		t.Error("直接子级 .x 应被 'A > B' 命中")
	}
	if isRedAt(img, 2) {
		t.Error("孙级 .x 不应被子代选择器命中")
	}
}

// TestCSSAdjacentSibling 验证相邻兄弟 "+":只命中紧随其后的兄弟。
func TestCSSAdjacentSibling(t *testing.T) {
	// 四个 rect 同级；.a + .x 只应命中紧跟 .a 的 idx1。
	body := `<rect class="a" x="0" y="0" width="10" height="10" fill="#0000ff"/>` +
		`<rect class="x" x="10" y="0" width="10" height="10" fill="#0000ff"/>` +
		`<rect class="x" x="20" y="0" width="10" height="10" fill="#0000ff"/>` +
		`<rect class="x" x="30" y="0" width="10" height="10" fill="#0000ff"/>`
	img := renderCombo(t, body, `.a + .x{fill:#ff0000}`, 40, 10)
	if !isRedAt(img, 1) {
		t.Error("紧随 .a 的 .x 应被相邻兄弟选择器命中")
	}
	if isRedAt(img, 2) || isRedAt(img, 3) {
		t.Error("非紧邻的 .x 不应被 '+' 命中")
	}
}

// TestCSSGeneralSibling 验证通用兄弟 "~":命中其后所有同级兄弟。
func TestCSSGeneralSibling(t *testing.T) {
	body := `<rect class="a" x="0" y="0" width="10" height="10" fill="#0000ff"/>` +
		`<rect class="x" x="10" y="0" width="10" height="10" fill="#0000ff"/>` +
		`<rect class="y" x="20" y="0" width="10" height="10" fill="#0000ff"/>` +
		`<rect class="x" x="30" y="0" width="10" height="10" fill="#0000ff"/>`
	img := renderCombo(t, body, `.a ~ .x{fill:#ff0000}`, 40, 10)
	if !isRedAt(img, 1) || !isRedAt(img, 3) {
		t.Error(".a 之后所有同级 .x 都应被 '~' 命中")
	}
	if isRedAt(img, 2) {
		t.Error(".y 不匹配 .x，不应被命中")
	}
}

// TestCSSCombinatorSpecificity 验证组合子各段的特异性累加：
// "#g .x"(id+class=(1,1,0)) 应胜过 ".outer .x"(class+class=(0,2,0))。
func TestCSSCombinatorSpecificity(t *testing.T) {
	body := `<g class="outer" id="g"><rect class="x" x="10" y="0" width="10" height="10" fill="#0000ff"/></g>`
	// 低特异性写红，高特异性写蓝；结果应为蓝。
	img := renderCombo(t, body, `.outer .x{fill:#ff0000}#g .x{fill:#0000ff}`, 40, 10)
	if isRedAt(img, 1) {
		t.Error("'#g .x' 特异性(1,1,0) 应胜过 '.outer .x'(0,2,0)，不应是红色")
	}
	c := img.NRGBAAt(15, 5)
	if !(c.B > 200 && c.R < 100) {
		t.Errorf("高特异性规则应填蓝, got %v", c)
	}
}

// TestCSSBackwardCompatSingleSelector 验证原有单段选择器行为不变（回归保护）。
func TestCSSBackwardCompatSingleSelector(t *testing.T) {
	body := `<rect class="x" x="10" y="0" width="10" height="10" fill="#0000ff"/>`
	img := renderCombo(t, body, `rect.x{fill:#ff0000}`, 40, 10)
	if !isRedAt(img, 1) {
		t.Error("单段复合选择器 rect.x 应照常命中")
	}
}
