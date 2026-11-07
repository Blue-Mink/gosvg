package gosvg

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
)

// node 是 SVG 元素树的一个节点。name 为元素名（小写），attrs 为属性表，
// content 是 <text> 这类元素的字符内容。inlineStyle 是内联 style 属性
// 解析结果（优先级高于 presentation 属性与 CSS 规则，在 resolve 阶段合并）。
type node struct {
	name        string
	attrs       map[string]string
	inlineStyle map[string]string
	children    []*node
	content     string
	parent      *node
}

// attr 返回属性值（区分大小写不敏感，已统一小写），缺失返回空串。
func (n *node) attr(key string) string {
	if n.attrs == nil {
		return ""
	}
	return n.attrs[strings.ToLower(key)]
}

// hasAttr 判断属性是否存在。
func (n *node) hasAttr(key string) bool {
	_, ok := n.attrs[strings.ToLower(key)]
	return ok
}

// parse 把 SVG 字节解析成元素树。解析失败返回 error。
func parse(data []byte) (*node, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var root *node
	var stack []*node

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &node{
				name:  strings.ToLower(t.Name.Local),
				attrs: make(map[string]string, len(t.Attr)),
			}
			for _, a := range t.Attr {
				n.attrs[strings.ToLower(a.Name.Local)] = a.Value
			}
			// 内联 style 属性（style="fill:red;...")优先级最高，单独保存，
			// 在 resolve 的 applyCSS 阶段最后覆盖 presentation 属性与 CSS 规则。
			if style := n.attrs["style"]; style != "" {
				n.inlineStyle = parseStyle(style)
			}
			if len(stack) == 0 {
				if root != nil {
					// 多根节点：只保留第一个，其余忽略。
					continue
				}
				root = n
			} else {
				parent := stack[len(stack)-1]
				n.parent = parent
				parent.children = append(parent.children, n)
			}
			stack = append(stack, n)

		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}

		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].content += string(t)
			}
		}
	}
	if root == nil {
		return nil, &UnsupportedError{Feature: "empty document", Code: CodeEmptyDocument, Err: ErrEmptyDocument}
	}
	return root, nil
}

// resolve 在解析后建立 defs/gradient 索引，并读取顶层 viewBox / width / height。
func (d *Document) resolve() error {
	d.defs = map[string]*node{}
	d.grads = map[string]*node{}

	if d.root.name == "svg" {
		if vb := d.root.attr("viewbox"); vb != "" {
			if v, ok := parseViewBox(vb); ok {
				d.viewBox = v
				d.hasVB = true
			}
		}
		d.width = parseLengthOr(d.root.attr("width"), 0)
		d.height = parseLengthOr(d.root.attr("height"), 0)
		d.par = parsePreserveAspectRatio(d.root.attr("preserveaspectratio"))
	}

	// 深度优先遍历，收集带 id 的元素（mask / gradient / 供 use 引用的形状）。
	// mask、linearGradient、radialGradient 常直接放在 <svg> 下而非 <defs> 里。
	var walk func(n *node)
	walk = func(n *node) {
		for _, c := range n.children {
			switch c.name {
			case "mask":
				if id := c.attr("id"); id != "" {
					d.defs[id] = c
				}
			case "lineargradient", "radialgradient":
				if id := c.attr("id"); id != "" {
					d.grads[id] = c
				}
			case "style":
				if c.content != "" {
					d.cssRules = append(d.cssRules, parseCSS(c.content)...)
				}
			case "defs":
				d.indexDefs(c)
			default:
				// 其余带 id 的元素（供 <use> 引用）也登记。
				if id := c.attr("id"); id != "" {
					if _, exists := d.defs[id]; !exists {
						d.defs[id] = c
					}
				}
			}
			walk(c)
		}
	}
	walk(d.root)

	// CSS 级联：把 <style> 规则与内联 style 合并进元素 attrs。
	d.applyCSS()

	return nil
}

// indexDefs 登记 defs 块内带 id 的元素。
func (d *Document) indexDefs(defsNode *node) {
	var walk func(n *node)
	walk = func(n *node) {
		for _, c := range n.children {
			if id := c.attr("id"); id != "" {
				d.defs[id] = c
			}
			if c.name == "lineargradient" || c.name == "radialgradient" {
				if id := c.attr("id"); id != "" {
					d.grads[id] = c
				}
			}
			walk(c)
		}
	}
	walk(defsNode)
}

// parseViewBox 解析 "min-x min-y width height"。
func parseViewBox(s string) ([4]float64, bool) {
	var v [4]float64
	fields := strings.Fields(s)
	if len(fields) != 4 {
		return v, false
	}
	ok := true
	for i, f := range fields {
		v[i], ok = parseFloat(f)
		if !ok {
			return v, false
		}
	}
	if v[2] <= 0 || v[3] <= 0 {
		return v, false
	}
	return v, true
}

// parseFloatOr 解析浮点，失败返回 def。
func parseFloatOr(s string, def float64) float64 {
	if v, ok := parseFloat(s); ok {
		return v
	}
	return def
}

// preserveAspect 保存解析后的 preserveAspectRatio：对齐位置（0/0.5/1）、
// slice 标志与 none（不保持比例）标志。
type preserveAspect struct {
	alignX, alignY float64
	slice          bool
	none           bool
}

// parsePreserveAspectRatio 解析 preserveAspectRatio 属性。
// 默认 xMidYMid meet；"none" 表示不保持比例（拉伸填充）。
func parsePreserveAspectRatio(s string) preserveAspect {
	p := preserveAspect{alignX: 0.5, alignY: 0.5}
	s = strings.TrimSpace(s)
	if s == "" {
		return p
	}
	fields := strings.Fields(s)
	switch fields[0] {
	case "none":
		p.none = true
		return p
	case "xMinYMin":
		p.alignX, p.alignY = 0, 0
	case "xMidYMin":
		p.alignX, p.alignY = 0.5, 0
	case "xMaxYMin":
		p.alignX, p.alignY = 1, 0
	case "xMinYMid":
		p.alignX, p.alignY = 0, 0.5
	case "xMidYMid":
		p.alignX, p.alignY = 0.5, 0.5
	case "xMaxYMid":
		p.alignX, p.alignY = 1, 0.5
	case "xMinYMax":
		p.alignX, p.alignY = 0, 1
	case "xMidYMax":
		p.alignX, p.alignY = 0.5, 1
	case "xMaxYMax":
		p.alignX, p.alignY = 1, 1
	}
	if len(fields) >= 2 {
		p.slice = fields[1] == "slice"
	}
	return p
}
