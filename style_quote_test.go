package gosvg

import "testing"

// TestParseStyleQuotedSemicolon 验证 parseStyle 按顶层分号切分声明，
// 不把引号内字符串或括号内内容里的分号当分隔符。
func TestParseStyleQuotedSemicolon(t *testing.T) {
	cases := []struct {
		name string
		in   string
		key  string
		want string
	}{
		{
			name: "引号内分号不切分",
			in:   `content:'a;b';fill:red`,
			key:  "content",
			want: "'a;b'",
		},
		{
			name: "引号内分号后的声明不丢",
			in:   `content:'a;b';fill:red`,
			key:  "fill",
			want: "red",
		},
		{
			name: "双引号内分号不切分",
			in:   `content:"a;b";stroke:blue`,
			key:  "content",
			want: `"a;b"`,
		},
		{
			name: "data URI 内分号不切分",
			in:   `fill:url(data:image/png;base64,AA);stroke:red`,
			key:  "fill",
			want: "url(data:image/png;base64,AA)",
		},
		{
			name: "data URI 后面的声明不丢",
			in:   `fill:url(data:image/png;base64,AA);stroke:red`,
			key:  "stroke",
			want: "red",
		},
		{
			name: "普通声明不受影响",
			in:   `fill:red;stroke:blue;stroke-width:2`,
			key:  "stroke-width",
			want: "2",
		},
		{
			name: "引号内的花括号不切分",
			in:   `content:'{';fill:green`,
			key:  "content",
			want: "'{'",
		},
		{
			name: "转义引号不提前结束字符串",
			in:   `content:'a\';b';fill:teal`,
			key:  "content",
			want: `'a\';b'`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseStyle(c.in)
			v, ok := got[c.key]
			if !ok {
				t.Fatalf("parseStyle(%q) 缺少键 %q，实际表 %v", c.in, c.key, got)
			}
			if v != c.want {
				t.Fatalf("parseStyle(%q)[%q] = %q，期望 %q", c.in, c.key, v, c.want)
			}
		})
	}
}

// TestParseStyleUnchangedSemantics 确认基础语义与旧实现一致，
// 防止引号感知改造引入回归。
func TestParseStyleUnchangedSemantics(t *testing.T) {
	got := parseStyle("fill:red;stroke:blue")
	if got["fill"] != "red" || got["stroke"] != "blue" {
		t.Fatalf("parseStyle 基础解析错误: %v", got)
	}
	if len(got) != 2 {
		t.Fatalf("期望 2 个属性，实际 %d 个: %v", len(got), got)
	}
	// 缺冒号 / 空声明 / 空值：静默跳过
	got = parseStyle(";;;nocolon;fill:;stroke:red;")
	if len(got) != 1 || got["stroke"] != "red" {
		t.Fatalf("非法声明应被静默跳过，实际: %v", got)
	}
	// key 小写化
	got = parseStyle("FILL:red")
	if got["fill"] != "red" {
		t.Fatalf("key 应小写化，实际: %v", got)
	}
}
