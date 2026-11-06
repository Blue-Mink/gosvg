package gosvg

import (
	"image/color"
	"strconv"
	"strings"

	"golang.org/x/image/colornames"
)

// parseFloat 解析 SVG 数字（支持 "3."、".5"、科学计数法等）。
func parseFloat(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// parseLength 解析带单位的长度值（px/pt/em/cm/mm/in/% 等）。
// 本库主要处理 viewBox 内的用户单位，非 px 单位按数值部分 1:1 近似。
func parseLength(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	// 剥离末尾的单位字母或百分号，只留数值部分。
	i := len(s)
	for i > 0 && (isUnitChar(s[i-1]) || s[i-1] == '%') {
		i--
	}
	return parseFloat(s[:i])
}

// isUnitChar 判断字节是否为长度单位字符（a-z / A-Z）。
func isUnitChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// parseLengthOr 解析带单位的长度，失败返回 def。
func parseLengthOr(s string, def float64) float64 {
	if v, ok := parseLength(s); ok {
		return v
	}
	return def
}

// parseStyle 解析 CSS 内联样式串 style="fill:red;stroke:blue;stroke-width:2"
// 为属性表（key 已小写）。空声明/缺冒号的部分静默跳过。
func parseStyle(s string) map[string]string {
	out := map[string]string{}
	for _, decl := range strings.Split(s, ";") {
		decl = strings.TrimSpace(decl)
		if decl == "" {
			continue
		}
		kv := strings.SplitN(decl, ":", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(kv[0]))
		v := strings.TrimSpace(kv[1])
		if k != "" && v != "" {
			out[k] = v
		}
	}
	return out
}

// parseColor 解析 SVG 颜色值，支持 #rgb / #rrggbb / #rgba / #rrggbbaa、
// rgb() / rgba()、SVG/CSS 命名颜色、none。透明颜色返回 ok=false（表示"不填充"）。
func parseColor(s string) (color.NRGBA, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "none" {
		return color.NRGBA{}, false
	}

	// 十六进制
	if strings.HasPrefix(s, "#") {
		hex := s[1:]
		switch len(hex) {
		case 3:
			return color.NRGBA{
				R: hexv(hex[0]) * 0x11,
				G: hexv(hex[1]) * 0x11,
				B: hexv(hex[2]) * 0x11,
				A: 0xff,
			}, true
		case 4:
			return color.NRGBA{
				R: hexv(hex[0]) * 0x11,
				G: hexv(hex[1]) * 0x11,
				B: hexv(hex[2]) * 0x11,
				A: hexv(hex[3]) * 0x11,
			}, true
		case 6:
			return color.NRGBA{
				R: hexv(hex[0])<<4 | hexv(hex[1]),
				G: hexv(hex[2])<<4 | hexv(hex[3]),
				B: hexv(hex[4])<<4 | hexv(hex[5]),
				A: 0xff,
			}, true
		case 8:
			return color.NRGBA{
				R: hexv(hex[0])<<4 | hexv(hex[1]),
				G: hexv(hex[2])<<4 | hexv(hex[3]),
				B: hexv(hex[4])<<4 | hexv(hex[5]),
				A: hexv(hex[6])<<4 | hexv(hex[7]),
			}, true
		}
		return color.NRGBA{}, false
	}

	// rgb() / rgba()
	if strings.HasPrefix(s, "rgb") {
		inner := s[strings.Index(s, "(")+1 : strings.LastIndex(s, ")")]
		parts := strings.Split(inner, ",")
		if len(parts) < 3 {
			return color.NRGBA{}, false
		}
		r, ok1 := parseColorComp(parts[0])
		g, ok2 := parseColorComp(parts[1])
		b, ok3 := parseColorComp(parts[2])
		a := 255
		if len(parts) >= 4 {
			af, ok4 := parseFloat(parts[3])
			if ok4 {
				a = int(clamp01(af) * 255)
			}
		}
		if !ok1 || !ok2 || !ok3 {
			return color.NRGBA{}, false
		}
		return color.NRGBA{R: r, G: g, B: b, A: uint8(a)}, true
	}

	// 命名颜色
	if c, ok := colornames.Map[s]; ok {
		return color.NRGBAModel.Convert(c).(color.NRGBA), true
	}

	return color.NRGBA{}, false
}

// parseColorComp 解析 rgb() 的一个分量（数字或百分比）。
func parseColorComp(s string) (uint8, bool) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		f, ok := parseFloat(strings.TrimSuffix(s, "%"))
		if !ok {
			return 0, false
		}
		return uint8(clamp01(f/100) * 255), true
	}
	f, ok := parseFloat(s)
	if !ok {
		return 0, false
	}
	return uint8(clamp01(f) * 255), true
}

func hexv(b byte) uint8 {
	switch {
	case b >= '0' && b <= '9':
		return b - '0'
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10
	}
	return 0
}

func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}
