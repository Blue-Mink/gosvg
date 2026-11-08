package gosvg

import "strings"

// parseViewportLength 解析 <svg> 的 width / height（视口尺寸）。
//
// 与元素几何用的 parseLengthOr 不同：视口的物理单位按 96dpi 换算成 px，
// 否则 width="193mm" 会被当成 193px，视口分母差 3.78 倍。注意全量对账显示
// 这项换算只救回两个 bamboo 文件（平均 IoU 0.9486 → 0.9507），**并不是**
// 上一版猜的「D 批 54 个低分文件的根因」；价值主要在语义正确性。
//
// 物理单位：px / mm / cm / in / pt / pc / q；无单位按 px。
// 相对单位与百分比（em/rem/ex/ch/vw/vh/vmin/vmax/%）在根 <svg> 上没有父参照，
// 保持「数值 1:1 近似」的旧行为（与 parseLength 一致），不做换算也不判非法。
// 未知后缀（如 12abc）判非法并返回 def。
func parseViewportLength(s string, def float64) float64 {
	t := strings.ToLower(strings.TrimSpace(s))
	if t == "" {
		return def
	}
	// 先剥离末尾的单位字母 / 百分号，再按单位分派，避免 "mm" 之类的多字母后缀
	// 被逐字符匹配抢先，也避免 "1e2px" 里的科学计数法被误当单位。
	i := len(t)
	for i > 0 && (isUnitChar(t[i-1]) || t[i-1] == '%') {
		i--
	}
	v, ok := parseFloat(strings.TrimSpace(t[:i]))
	if !ok {
		return def
	}
	switch t[i:] {
	case "", "px":
		return v
	case "mm":
		return v * 96.0 / 25.4
	case "cm":
		return v * 96.0 / 2.54
	case "in":
		return v * 96
	case "pt":
		return v * 96.0 / 72
	case "pc":
		return v * 16
	case "q":
		return v * 96.0 / 101.6
	case "%", "em", "rem", "ex", "ch", "vw", "vh", "vmin", "vmax":
		// 根 <svg> 上无父参照 / 视口基准，沿用旧的「数值 1:1 近似」。
		return v
	default:
		return def
	}
}
