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
func parseCSSLimited(text string) (rules []cssRule, truncated bool) {
	for _, block := range strings.Split(stripCSSComments(text), "}") {
		if len(rules) >= MaxCSSRules {
			return rules, true
		}
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		parts := strings.SplitN(block, "{", 2)
		if len(parts) != 2 {
			continue
		}
		if len(parts[1]) > MaxCSSDeclLen {
			truncated = true
			continue
		}
		decls := parseStyle(parts[1]) // 复用内联样式解析（按 ";" 分割，key:value）
		if len(decls) == 0 {
			continue
		}
		rule := cssRule{decls: decls}
		for _, sel := range strings.Split(parts[0], ",") {
			sel = strings.TrimSpace(sel)
			if sel == "" {
				continue
			}
			if len(sel) > MaxCSSSelectorLen {
				truncated = true
				continue
			}
			rule.selectors = append(rule.selectors, parseSelector(sel))
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
