package gosvg

import (
	"sort"
	"strings"
)

// cssCompound 表示一个复合选择器：类型、class 列表、id、通用选择器的任意组合
// （如 "rect.foo.bar#baz"、"*"）。
type cssCompound struct {
	tag       string
	classes   []string
	id        string
	universal bool // "*" 或空选择器段的通用匹配
}

// cssCombinator 描述相邻两个复合选择器之间的关系。
type cssCombinator byte

const (
	combDescendant cssCombinator = ' ' // 后代：A B（B 是 A 的任意后代）
	combChild      cssCombinator = '>' // 子代：A > B
	combAdjacent   cssCombinator = '+' // 相邻兄弟：A + B
	combSibling    cssCombinator = '~' // 通用兄弟：A ~ B
)

// cssSelector 表示一条完整选择器：若干复合选择器用组合子连接，
// 从右往左匹配（最右段是目标元素）。单段即为原来的简单选择器，行为完全兼容。
// 伪类/伪元素仍不支持（解析时被当作普通段名，不会匹配任何元素）。
type cssSelector struct {
	compounds   []cssCompound
	combinators []cssCombinator // len = len(compounds)-1，combinators[i] 连接 compounds[i] 与 compounds[i+1]
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

// parseCompound 解析一个复合选择器片段（tag#id.class 的任意组合，或 "*"）。
func parseCompound(s string) cssCompound {
	var c cssCompound
	i := 0
	for i < len(s) && s[i] != '#' && s[i] != '.' {
		i++
	}
	c.tag = strings.TrimSpace(s[:i])
	if c.tag == "*" {
		c.tag = ""
		c.universal = true
	}
	for i < len(s) {
		switch s[i] {
		case '#':
			i++
			j := i
			for j < len(s) && s[j] != '#' && s[j] != '.' {
				j++
			}
			c.id = s[i:j]
			i = j
		case '.':
			i++
			j := i
			for j < len(s) && s[j] != '#' && s[j] != '.' {
				j++
			}
			c.classes = append(c.classes, s[i:j])
			i = j
		}
	}
	return c
}

// parseSelector 解析一条选择器：按组合子（空格 / > / + / ~）切成若干复合段，
// 一遍扫描同时产出 compounds 与 combinators（combinators[k] 连接 compounds[k] 与 [k+1]）。
// 连续空白、组合子两侧空白都被折叠；伪类里的括号内容当前不做特殊处理，
// 会留在段文本中从而不匹配任何元素（明确不支持伪类，见文档）。
func parseSelector(s string) cssSelector {
	var sel cssSelector
	var buf strings.Builder
	// flush 收尾当前复合段。pending 是等待确认的组合子（读到下一段时才真正落位）。
	var pending cssCombinator
	havePending := false
	flush := func() {
		part := strings.TrimSpace(buf.String())
		buf.Reset()
		if part == "" {
			return
		}
		if len(sel.compounds) > 0 && havePending {
			sel.combinators = append(sel.combinators, pending)
		}
		sel.compounds = append(sel.compounds, parseCompound(part))
		havePending = false
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '>', '+', '~':
			if strings.TrimSpace(buf.String()) != "" {
				flush() // 组合子前的段已完整
			}
			pending = cssCombinator(ch)
			havePending = true
		case ' ', '\t', '\n', '\r':
			j := i
			for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
				j++
			}
			// 空白后紧跟显式组合子：空白本身不算组合子，交给下一轮处理。
			if j < len(s) && (s[j] == '>' || s[j] == '+' || s[j] == '~') {
				i = j - 1
				continue
			}
			if strings.TrimSpace(buf.String()) != "" {
				flush()
				pending = combDescendant
				havePending = true
			}
			i = j - 1
		default:
			buf.WriteByte(ch)
		}
	}
	flush()
	return sel
}

// compoundMatches 判断复合选择器 c 是否匹配元素 n。
func compoundMatches(c cssCompound, n *node) bool {
	if c.tag != "" && c.tag != n.name {
		return false
	}
	if c.id != "" && n.attr("id") != c.id {
		return false
	}
	if len(c.classes) > 0 {
		classSet := make(map[string]bool, 4)
		for _, cl := range strings.Fields(n.attr("class")) {
			classSet[cl] = true
		}
		for _, cl := range c.classes {
			if !classSet[cl] {
				return false
			}
		}
	}
	return true
}

// siblingBefore 返回 n 之前最近的同父元素（忽略非元素节点，这里 children 全是元素）。
func siblingBefore(n *node) *node {
	if n == nil || n.parent == nil {
		return nil
	}
	for i, c := range n.parent.children {
		if c == n {
			if i == 0 {
				return nil
			}
			return n.parent.children[i-1]
		}
	}
	return nil
}

// matches 判断元素 n 是否匹配整条选择器。从最右复合段开始，沿组合子链向左回溯。
func (sel cssSelector) matches(n *node) bool {
	if len(sel.compounds) == 0 {
		return false
	}
	// 从右往左：cur 是当前要匹配的候选元素，ci 是复合段下标。
	ci := len(sel.compounds) - 1
	cur := n
	if !compoundMatches(sel.compounds[ci], cur) {
		return false
	}
	for ci > 0 {
		comb := sel.combinators[ci-1]
		switch comb {
		case combChild:
			cur = cur.parent
		case combDescendant:
			cur = cur.parent
			for cur != nil && !compoundMatches(sel.compounds[ci-1], cur) {
				cur = cur.parent
			}
			if cur == nil {
				return false
			}
			ci--
			continue
		case combAdjacent:
			cur = siblingBefore(cur)
		case combSibling:
			cur = siblingBefore(cur)
			for cur != nil && !compoundMatches(sel.compounds[ci-1], cur) {
				cur = siblingBefore(cur)
			}
			if cur == nil {
				return false
			}
			ci--
			continue
		default:
			return false
		}
		if cur == nil || !compoundMatches(sel.compounds[ci-1], cur) {
			return false
		}
		ci--
	}
	return true
}

// specificity 返回 (id, class, tag) 三元组，按 CSS 标准累加各复合段。
func (sel cssSelector) specificity() (id, class, tag int) {
	for _, c := range sel.compounds {
		if c.tag != "" {
			tag++
		}
		if c.id != "" {
			id++
		}
		class += len(c.classes)
	}
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
