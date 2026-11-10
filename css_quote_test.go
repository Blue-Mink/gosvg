package gosvg

import "testing"

// TestCSSQuotedBraces 守住 css.go 的引号感知扫描：CSS 字符串里的 { } 不是语法括号，
// 旧实现会把 content:"{" 之后的整条规则吞进声明值、把 content:"}" 的声明体截断。
func TestCSSQuotedBraces(t *testing.T) {
	cases := []struct {
		name string
		css  string
		want int
	}{
		{"plain", `.a{fill:red} .b{fill:blue}`, 2},
		{"at-rule skipped", `@media screen { .x{display:none} } .ok{fill:green}`, 1},
		{"quoted open brace", `.c{content:"{"} .d{fill:red}`, 2},
		{"quoted close brace", `.g{content:"}"} .h{fill:red}`, 2},
		{"escaped quote", `.i{content:"a\"}"} .j{fill:red}`, 2},
		{"selector attribute", `[data-x="a;b"]{fill:red} .k{fill:blue}`, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rules, _ := parseCSSLimited(tc.css)
			if len(rules) != tc.want {
				t.Fatalf("规则数 %d，期望 %d；rules=%+v", len(rules), tc.want, rules)
			}
		})
	}
	// 额外确认被修复的那条规则真的解析出了目标声明，而不只是数量对上。
	rules, _ := parseCSSLimited(`.c{content:"{"} .d{fill:red}`)
	found := false
	for _, r := range rules {
		if v, ok := r.decls["fill"]; ok && v == "red" {
			found = true
		}
	}
	if !found {
		t.Error(`.d{fill:red} 未解析出来：引号内括号仍吞掉了后续规则`)
	}
}
