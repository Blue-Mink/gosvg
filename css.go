package gosvg

import (
	"sort"
	"strings"
)

// cssSelector 表示一个简单选择器：类型、class 列表、id 的任意组合
// （如 "rect.foo.bar#baz"）。不支持后代/子代/伪类选择器。
type cssSelector struct {
	tag     string
	classes []string
	id      string
}

// cssRule 表示一条 CSS 规则：选择器（逗号分隔展开）+ 声明。
type cssRule struct {
	selectors []cssSelector
	decls     map[string]string
}

// stripCSSComments 去除 /* ... */ 注释。
func stripCSSComments(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, "/*")
		if i < 0 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:i])
		s = s[i+2:]
		if j := strings.Index(s, "*/"); j >= 0 {
			s = s[j+2:]
		} else {
			break
		}
	}
	return b.String()
}

// CSS 解析规模上限：防止超大 <style> 或超多规则在解析阶段耗尽内存。
// 超限内容被丢弃（不是让整个渲染失败），并通过 parseCSSLimited 的返回值上报。
const (
	// MaxCSSRules 是单个 <style> 元素解析出的规则条数上限，超出部分丢弃。
	MaxCSSRules = 10000
	// MaxCSSSelectorLen 是单个选择器文本长度上限，超长选择器跳过。
	MaxCSSSelectorLen = 512
	// MaxCSSDeclLen 是单条规则声明块长度上限，超长声明块跳过。
	MaxCSSDeclLen = 4096
)

// parseCSS 解析 <style> 元素的 CSS 文本为规则列表（超限部分静默丢弃）。
func parseCSS(text string) []cssRule {
	rules, _ := parseCSSLimited(text)
	return rules
}

// parseCSSLimited 同 parseCSS，并报告是否因超出 MaxCSSRules / MaxCSSSelectorLen /
// MaxCSSDeclLen 而丢弃了内容，供调用方记入 Diagnostics。
// skipCSSQuoted 从 text[i] 处的引号开始，返回闭合引号之后的下标。
// CSS 字符串里的 { } ; 都不是语法字符：旧实现按裸字节数括号，遇到
// content:"{" 会把后续规则整条吞进声明值，遇到 content:"}" 会把声明体截断。
// 支持反斜杠转义；未闭合时返回 len(text)，交由调用方按「未闭合」处理。
func skipCSSQuoted(text string, i int) int {
	q := text[i]
	for i++; i < len(text); {
		if text[i] == '\\' {
			i += 2
			continue
		}
		if text[i] == q {
			return i + 1
		}
		i++
	}
	return len(text)
}

func findCSSRuleOpen(text string, from int) int {
	for i := from; i < len(text); {
		switch text[i] {
		case '\'', '"':
			i = skipCSSQuoted(text, i)
		case '{':
			return i
		default:
			i++
		}
	}
	return -1
}

func findCSSRuleClose(text string, from int) int {
	depth := 1
	for i := from; i < len(text); {
		switch text[i] {
		case '\'', '"':
			i = skipCSSQuoted(text, i)
			continue
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return -1
}

// lastCSSTopLevelSep 找 sel 里最后一个顶层 ; 或 }（跳过属性选择器等引号内容），
// 用于剥掉游离的 } 与无块 at-statement 前缀。
func lastCSSTopLevelSep(sel string) int {
	last := -1
	for i := 0; i < len(sel); {
		switch sel[i] {
		case '\'', '"':
			i = skipCSSQuoted(sel, i)
			continue
		case ';', '}':
			last = i
		}
		i++
	}
	return last
}

func parseCSSLimited(text string) (rules []cssRule, truncated bool) {
	text = stripCSSComments(text)
	// 按大括号深度扫描顶层规则。不能用 strings.Split(text, "}") 切块：
	// @media / @keyframes / @supports 等 at-rule 内部含嵌套的 {}，
	// 简单切分会把嵌套里的 "#id{display:none}" 当成顶层规则应用，
	// 在默认媒体下错误隐藏元素（真实语料 D__altText_spinner 的沙漏整块
	// 消失，gosvg 少画 18661 px）。gosvg 不求值媒体查询，整块跳过
	// at-rule，与 cairosvg 的行为一致。所有扫描都跳过引号内字符串。
	for pos := 0; pos < len(text); {
		open := findCSSRuleOpen(text, pos)
		if open < 0 {
			break
		}
		sel := text[pos:open]
		// 顶层游离的 '}' 或无块 at-statement（@import ...;）之后的才是选择器。
		if k := lastCSSTopLevelSep(sel); k >= 0 {
			sel = sel[k+1:]
		}
		// 深度感知地找到与 open 匹配的 '}'。
		close := findCSSRuleClose(text, open+1)
		var body string
		var end int
		if close < 0 { // 未闭合：保留剩余声明并终止扫描
			body = text[open+1:]
			end = len(text)
		} else {
			body = text[open+1 : close]
			end = close + 1
		}
		pos = end
		sel = strings.TrimSpace(sel)
		if sel == "" || strings.HasPrefix(sel, "@") {
			continue
		}
		if len(rules) >= MaxCSSRules {
			return rules, true
		}
		if len(body) > MaxCSSDeclLen {
			truncated = true
			continue
		}
		decls := parseStyle(body) // 复用内联样式解析（按 ";" 分割，key:value）
		if len(decls) == 0 {
			continue
		}
		rule := cssRule{decls: decls}
		for _, s := range strings.Split(sel, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if len(s) > MaxCSSSelectorLen {
				truncated = true
				continue
			}
			rule.selectors = append(rule.selectors, parseSelector(s))
		}
		if len(rule.selectors) > 0 {
			rules = append(rules, rule)
		}
	}
	return rules, truncated
}

// parseSelector 解析单个简单选择器（tag#id.class 的任意组合）。
func parseSelector(s string) cssSelector {
	var sel cssSelector
	i := 0
	for i < len(s) && s[i] != '#' && s[i] != '.' {
		i++
	}
	sel.tag = strings.TrimSpace(s[:i])
	for i < len(s) {
		switch s[i] {
		case '#':
			i++
			j := i
			for j < len(s) && s[j] != '#' && s[j] != '.' {
				j++
			}
			sel.id = s[i:j]
			i = j
		case '.':
			i++
			j := i
			for j < len(s) && s[j] != '#' && s[j] != '.' {
				j++
			}
			sel.classes = append(sel.classes, s[i:j])
			i = j
		}
	}
	return sel
}

// matches 判断元素 n 是否匹配选择器。
func (sel cssSelector) matches(n *node) bool {
	if sel.tag != "" && sel.tag != n.name {
		return false
	}
	if sel.id != "" && n.attr("id") != sel.id {
		return false
	}
	if len(sel.classes) > 0 {
		classSet := make(map[string]bool, 4)
		for _, c := range strings.Fields(n.attr("class")) {
			classSet[c] = true
		}
		for _, c := range sel.classes {
			if !classSet[c] {
				return false
			}
		}
	}
	return true
}

// specificity 返回 (id, class, tag) 三元组，用于 CSS 特异性比较。
func (sel cssSelector) specificity() (id, class, tag int) {
	if sel.tag != "" {
		tag = 1
	}
	if sel.id != "" {
		id = 1
	}
	class = len(sel.classes)
	return
}

// applyCSS 遍历元素树，把匹配的 CSS 规则声明与内联 style 合并进 attrs。
// 优先级：presentation 属性 < CSS 规则（按特异性）< 内联 style。
func (d *Document) applyCSS() {
	type match struct {
		rule cssRule
		sel  cssSelector
	}
	var walk func(n *node)
	walk = func(n *node) {
		// 收集匹配的 (规则, 选择器)，按特异性从低到高排序。
		var matches []match
		for _, rule := range d.cssRules {
			for _, sel := range rule.selectors {
				if sel.matches(n) {
					matches = append(matches, match{rule: rule, sel: sel})
				}
			}
		}
		sort.SliceStable(matches, func(i, j int) bool {
			ia, ib, ic := matches[i].sel.specificity()
			ja, jb, jc := matches[j].sel.specificity()
			if ia != ja {
				return ia < ja
			}
			if ib != jb {
				return ib < jb
			}
			return ic < jc
		})
		// 低特异性先写，高特异性后写覆盖；内联 style 始终优先。
		for _, m := range matches {
			for k, v := range m.rule.decls {
				if _, hasInline := n.inlineStyle[k]; hasInline {
					continue
				}
				n.attrs[k] = v
			}
		}
		for k, v := range n.inlineStyle {
			n.attrs[k] = v
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(d.root)
}
