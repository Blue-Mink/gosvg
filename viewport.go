package gosvg

import "strings"

// parseViewportLength 解析 <svg> 的 width / height（视口尺寸）。
//
// 与元素几何用的 parseLengthOr 不同：视口的物理单位必须按 96dpi 换算成 px，
// 否则 width="193mm" 会被当成 193px，视口分母差 3.78 倍，内容被错误地缩放/裁切
// （真实语料 D 批 54 个低 IoU 文件的根因）。
//
// 支持：px / mm / cm / in / pt / pc / q；无单位按 px；相对单位（em/rem/%/ex 等）
// 在根 <svg> 上没有父参照，保持「数值 1:1 近似」的旧行为，不做换算。
func parseViewportLength(s string, def float64) float64 {
	t := strings.ToLower(strings.TrimSpace(s))
	if t == "" {
		return def
	}
	// 按后缀长度降序匹配，避免 "mm" 被 "m" 抢先；px 必须排在 q/pt/pc 之后按精确后缀判断。
	// 百分比在根 <svg> 上没有父参照，保持「数值 1:1 近似」的旧行为。
	if strings.HasSuffix(t, "%") {
		v, ok := parseFloat(strings.TrimSpace(t[:len(t)-1]))
		if !ok {
			return def
		}
		return v
	}
	for _, u := range []struct {
		suffix string
		px     float64
	}{
		{"px", 1},
		{"mm", 96.0 / 25.4},
		{"cm", 96.0 / 2.54},
		{"in", 96},
		{"pt", 96.0 / 72},
		{"pc", 16},
		{"q", 96.0 / 101.6},
	} {
		if strings.HasSuffix(t, u.suffix) {
			v, ok := parseFloat(strings.TrimSpace(t[:len(t)-len(u.suffix)]))
			if !ok {
				return def
			}
			return v * u.px
		}
	}
	v, ok := parseFloat(t)
	if !ok {
		return def
	}
	return v
}
