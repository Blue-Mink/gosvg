package gosvg

import "testing"

// TestCSSAtRuleNestedRulesIgnored 验证 @media / @keyframes 等 at-rule 内部的
// 嵌套规则不会被当成顶层规则应用。修复前 parseCSSLimited 按 "}" 暴力切块，
// 会把 @media 里的 "#x{display:none}" 提升为顶层规则，在默认媒体下错误隐藏
// 元素（真实语料 D__altText_spinner 的沙漏整块消失，IoU 0.4549）。
// at-rule 之外的顶层规则仍必须正常生效。
func TestCSSAtRuleNestedRulesIgnored(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 10 10"><style>
    @media not (prefers-reduced-motion) {
      @keyframes spin { from { rotate: 0; } to { rotate: 360deg } }
      #hidden { display: none; }
    }
    .shown { fill: #ff0000; }
  </style>
  <g id="hidden"><rect x="0" y="0" width="5" height="10" fill="#0000ff"/></g>
  <rect class="shown" x="5" y="0" width="5" height="10" fill="#00ff00"/>
</svg>`)
	img, err := Render(data, 10)
	if err != nil {
		t.Fatal(err)
	}
	if a := img.NRGBAAt(1, 5).A; a == 0 {
		t.Error("@media 内的 #hidden{display:none} 不该在默认媒体下生效")
	}
	if c := img.NRGBAAt(7, 5); c.R < 200 || c.G > 60 {
		t.Errorf("at-rule 外的顶层 .shown{fill:red} 仍应生效, got %v", c)
	}
}
