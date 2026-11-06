package gosvg

import (
	"math"
	"strings"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
)

// parseTransform 解析 SVG transform 属性为 rasterx.Matrix2D。
// 支持 matrix / translate / scale / rotate / skewX / skewY，可空格/逗号分隔组合。
// 解析失败的部分静默跳过（返回单位矩阵那一段）。
func parseTransform(s string) rasterx.Matrix2D {
	m := rasterx.Identity
	s = strings.TrimSpace(s)
	for s != "" {
		s = strings.TrimLeft(s, " ,")
		if s == "" {
			break
		}
		end := strings.IndexByte(s, ')')
		if end < 0 {
			break
		}
		head := s[:end+1]
		s = s[end+1:]

		nameEnd := strings.IndexByte(head, '(')
		if nameEnd < 0 {
			continue
		}
		name := strings.TrimSpace(head[:nameEnd])
		args := parseNumList(head[nameEnd+1 : len(head)-1])

		switch name {
		case "matrix":
			if len(args) >= 6 {
				m = m.Mult(rasterx.Matrix2D{A: args[0], B: args[1], C: args[2], D: args[3], E: args[4], F: args[5]})
			}
		case "translate":
			tx := argOr(args, 0)
			ty := argOr(args, 1)
			m = m.Mult(rasterx.Identity.Translate(tx, ty))
		case "scale":
			sx := argOr(args, 0)
			sy := sx
			if len(args) >= 2 {
				sy = args[1]
			}
			m = m.Mult(rasterx.Identity.Scale(sx, sy))
		case "rotate":
			a := argOr(args, 0) * math.Pi / 180
			if len(args) >= 3 {
				// rotate(a, cx, cy) = translate(cx,cy) rotate(a) translate(-cx,-cy)
				cx, cy := args[1], args[2]
				m = m.Mult(rasterx.Identity.Translate(cx, cy)).
					Mult(rasterx.Identity.Rotate(a)).
					Mult(rasterx.Identity.Translate(-cx, -cy))
			} else {
				m = m.Mult(rasterx.Identity.Rotate(a))
			}
		case "skewx":
			m = m.Mult(rasterx.Identity.SkewX(argOr(args, 0) * math.Pi / 180))
		case "skewy":
			m = m.Mult(rasterx.Identity.SkewY(argOr(args, 0) * math.Pi / 180))
		}
	}
	return m
}

func argOr(args []float64, i int) float64 {
	if i < len(args) {
		return args[i]
	}
	return 0
}

// parseNumList 解析逗号/空格分隔的数字列表（忽略多余的逗号）。
func parseNumList(s string) []float64 {
	var out []float64
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	for _, f := range fields {
		if v, ok := parseFloat(f); ok {
			out = append(out, v)
		}
	}
	return out
}
