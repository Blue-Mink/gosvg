package gosvg

import (
	"image/color"
	"math"
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
	// NaN / ±Inf 会让坐标、线宽与矩阵运算失去意义，一律拒绝。
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

// svgLengthUnits 是允许出现在长度数值后的单位后缀。
// 本库只处理 viewBox 内的用户单位：所有单位统一剥离后按数值部分 1:1 近似，
// 不做物理单位换算（12pt 按 12 处理），% 也不按父值/视口基准解析。
// 例外：根 <svg> 的 width/height 是视口尺寸，按 96dpi 换算物理单位，见 viewport.go。
var svgLengthUnits = map[string]bool{
	"px": true, "pt": true, "pc": true, "cm": true, "mm": true, "in": true,
	"em": true, "ex": true, "rem": true, "q": true, "ch": true,
	"vw": true, "vh": true, "vmin": true, "vmax": true,
}

// parseLength 解析带单位的长度值；未知后缀（如 "12abc"）视为非法。
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
	if unit := strings.ToLower(s[i:]); unit != "" && unit != "%" {
		if !svgLengthUnits[unit] {
			return 0, false
		}
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
		if !isHexDigits(hex) {
			// 非法十六进制字符（如 #gggggg）是无效颜色：按 CSS 语义应忽略该声明，
			// 而不是被 hexv 静默当成 0 渲染成黑色。
			return color.NRGBA{}, false
		}
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

	// rgb() / rgba() / hsl() / hsla()
	if strings.HasPrefix(s, "rgb") || strings.HasPrefix(s, "hsl") {
		return parseColorFunc(s)
	}

	// 命名颜色
	if c, ok := colornames.Map[s]; ok {
		return color.NRGBAModel.Convert(c).(color.NRGBA), true
	}

	return color.NRGBA{}, false
}

// parseColorFunc 解析 rgb()/rgba()/hsl()/hsla() 颜色函数。
// 支持逗号分隔与空格分隔（含 "/" 分隔的 alpha，如 "rgb(255 0 0 / 0.5)"）。
func parseColorFunc(s string) (color.NRGBA, bool) {
	open := strings.IndexByte(s, '(')
	close := strings.LastIndexByte(s, ')')
	if open < 0 || close < open {
		return color.NRGBA{}, false
	}
	name := strings.TrimSpace(s[:open])
	inner := s[open+1 : close]

	// 分离斜杠语法 alpha（如 "255 0 0 / 0.5"）。
	alpha := 1.0
	if i := strings.IndexByte(inner, '/'); i >= 0 {
		if af, ok := parseAlpha(inner[i+1:]); ok {
			alpha = af
		}
		inner = inner[:i]
	}

	// 分量：优先逗号，其次空白。
	var comps []string
	if strings.Contains(inner, ",") {
		comps = strings.Split(inner, ",")
	} else {
		comps = strings.Fields(inner)
	}

	switch name {
	case "rgb", "rgba":
		if len(comps) < 3 {
			return color.NRGBA{}, false
		}
		r, ok1 := parseColorComp(comps[0])
		g, ok2 := parseColorComp(comps[1])
		b, ok3 := parseColorComp(comps[2])
		if !ok1 || !ok2 || !ok3 {
			return color.NRGBA{}, false
		}
		// 逗号分隔的第 4 分量（rgba 老语法）优先于斜杠 alpha。
		if name == "rgba" && len(comps) >= 4 {
			if af, ok := parseAlpha(comps[3]); ok {
				alpha = af
			}
		}
		return color.NRGBA{R: r, G: g, B: b, A: uint8(alpha * 255)}, true

	case "hsl", "hsla":
		if len(comps) < 3 {
			return color.NRGBA{}, false
		}
		h, ok1 := parseHue(comps[0])
		sat, ok2 := parsePercent(comps[1])
		lig, ok3 := parsePercent(comps[2])
		if !ok1 || !ok2 || !ok3 {
			return color.NRGBA{}, false
		}
		r, g, b := hslToRGB(h, sat, lig)
		if name == "hsla" && len(comps) >= 4 {
			if af, ok := parseAlpha(comps[3]); ok {
				alpha = af
			}
		}
		return color.NRGBA{R: r, G: g, B: b, A: uint8(alpha * 255)}, true
	}
	return color.NRGBA{}, false
}

// parseAlpha 解析 alpha 分量（数字 0..1 或百分比 0%..100%）。
func parseAlpha(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		f, ok := parseFloat(strings.TrimSuffix(s, "%"))
		if !ok {
			return 0, false
		}
		return clamp01(f / 100), true
	}
	f, ok := parseFloat(s)
	if !ok {
		return 0, false
	}
	return clamp01(f), true
}

// parseHue 解析色相（度，支持 deg 后缀），归一化到 [0,360)。
func parseHue(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "deg") {
		s = strings.TrimSuffix(s, "deg")
	}
	f, ok := parseFloat(s)
	if !ok {
		return 0, false
	}
	f = math.Mod(f, 360)
	if f < 0 {
		f += 360
	}
	return f, true
}

// parsePercent 解析百分比分量（"50%" → 0.5，或数字 0..1）。
func parsePercent(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		f, ok := parseFloat(strings.TrimSuffix(s, "%"))
		if !ok {
			return 0, false
		}
		return clamp01(f / 100), true
	}
	f, ok := parseFloat(s)
	if !ok {
		return 0, false
	}
	return clamp01(f), true
}

// hslToRGB 把 HSL（h 0..360，s/l 0..1）转成 RGB（0..255）。
func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	if s == 0 {
		v := uint8(l*255 + 0.5)
		return v, v, v
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	return hueToRGB(p, q, h/360+1.0/3.0),
		hueToRGB(p, q, h/360),
		hueToRGB(p, q, h/360-1.0/3.0)
}

// hueToRGB 把 hue 分量（0..1，已含偏移）转成 0..255。
func hueToRGB(p, q, t float64) uint8 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6.0:
		return uint8((p + (q-p)*6*t) * 255)
	case t < 1.0/2.0:
		return uint8(q * 255)
	case t < 2.0/3.0:
		return uint8((p + (q-p)*(2.0/3.0-t)*6) * 255)
	default:
		return uint8(p * 255)
	}
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

// isHexDigits 判断字符串是否非空且全为十六进制字符。
func isHexDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}
