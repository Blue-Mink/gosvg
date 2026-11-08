package gosvg

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// CharsetReader 供 encoding/xml 处理非 UTF-8 的编码声明，可由调用方替换以扩展
// 支持的编码（例如用 golang.org/x/text/encoding/htmlindex 覆盖全部 IANA 编码）。
//
// 这是包级变量，应在渲染前设置一次、之后只读；置 nil 表示禁用编码转换
// （此时非 UTF-8 声明会像 encoding/xml 默认行为一样报错）。
var CharsetReader func(charset string, input io.Reader) (io.Reader, error) = defaultCharsetReader

// defaultCharsetReader 是零外部依赖的默认实现，支持：
//
//	UTF-8 / UTF8            —— 直接透传
//	ASCII 系（US-ASCII 等）  —— 字节流本就是合法 UTF-8，直接透传
//	ISO-8859-1 / Latin-1    —— 单字节逐个展开为 UTF-8
//
// 其余编码返回带编码名的明确错误，便于定位（而不是 encoding/xml 那句
// "Decoder.CharsetReader is nil"）。真实语料实测：一批 259 个混合来源 SVG 中
// 有 67 个（25.9%）仅因 encoding="ASCII" 声明而 Parse 失败。
func defaultCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(charset)) {
	case "", "utf-8", "utf8":
		return input, nil
	case "ascii", "us-ascii", "us", "iso-ir-6", "iso646-us",
		"ansi_x3.4-1968", "ansi_x3.4-1986":
		return input, nil // ASCII 是 UTF-8 的真子集
	case "iso-8859-1", "iso8859-1", "latin1", "latin-1", "l1", "cp819", "ibm819":
		return decodeLatin1(input)
	}
	return nil, fmt.Errorf("gosvg: unsupported character encoding %q", charset)
}

// decodeLatin1 把 ISO-8859-1 单字节流展开为 UTF-8。
// 一次性读取（输入已受 MaxSVGSize 限制），避免流式转换的状态机复杂度。
func decodeLatin1(r io.Reader) (io.Reader, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxSVGSize))
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		if c < 0x80 {
			out = append(out, c)
			continue
		}
		out = append(out, 0xc0|c>>6, 0x80|c&0x3f)
	}
	return bytes.NewReader(out), nil
}
