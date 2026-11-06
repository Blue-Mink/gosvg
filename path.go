package gosvg

import (
	"math"
	"strings"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
	"golang.org/x/image/math/fixed"
)

// fpt 把浮点坐标转成 fixed.Point26_6（四舍五入；截断会引入 -x/-y 方向性偏移）。
func fpt(x, y float64) fixed.Point26_6 {
	return fixed.Point26_6{X: fixed.Int26_6(x*64 + 0.5), Y: fixed.Int26_6(y*64 + 0.5)}
}

// parsePathData 解析 SVG path 的 `d` 属性为 rasterx.Path。
// 支持全部指令：M/L/H/V/C/S/Q/T/A/Z（含相对小写形式）。
func parsePathData(d string) rasterx.Path {
	var p rasterx.Path
	var cx, cy float64   // 当前点
	var sx, sy float64   // 子路径起点
	var lcx, lcy float64 // 上一个控制点（S/T 反射用）
	var lastCmd byte

	fields := tokenizePath(d)
	i := 0
	for i < len(fields) {
		f := fields[i]
		i++

		// 指令字母
		if isCmd(f) {
			lastCmd = f[0]
		} else if lastCmd == 0 {
			// 开头没有指令，跳过
			continue
		} else {
			// 数字，复用上一个指令
			i--
		}

		cmd := lastCmd
		upper := cmd >= 'A' && cmd <= 'Z'
		cc := cmd
		if !upper && cc >= 'a' && cc <= 'z' {
			cc = cc - 'a' + 'A'
		}

		// 取该指令需要的参数数量
		n := argCount(cc)
		args := takeArgs(fields, &i, n)
		if len(args) < n {
			break
		}

		// 相对坐标偏移
		off := func(x, y float64) (float64, float64) {
			if upper {
				return x, y
			}
			return x + cx, y + cy
		}

		switch cc {
		case 'M':
			nx, ny := off(args[0], args[1])
			cx, cy = nx, ny
			sx, sy = nx, ny
			p.Start(fpt(cx, cy))
			// 隐式 LineTo：M 后续的坐标对按直线段处理
			for i < len(fields) && !isCmd(fields[i]) {
				sub := takeArgs(fields, &i, 2)
				if len(sub) < 2 {
					break
				}
				lx, ly := off(sub[0], sub[1])
				p.Line(fpt(lx, ly))
				cx, cy = lx, ly
			}
		case 'L':
			nx, ny := off(args[0], args[1])
			p.Line(fpt(nx, ny))
			cx, cy = nx, ny
		case 'H':
			nx := args[0]
			if !upper {
				nx += cx
			}
			p.Line(fpt(nx, cy))
			cx = nx
		case 'V':
			ny := args[0]
			if !upper {
				ny += cy
			}
			p.Line(fpt(cx, ny))
			cy = ny
		case 'C':
			x1, y1 := off(args[0], args[1])
			x2, y2 := off(args[2], args[3])
			nx, ny := off(args[4], args[5])
			p.CubeBezier(fpt(x1, y1), fpt(x2, y2), fpt(nx, ny))
			lcx, lcy = x2, y2
			cx, cy = nx, ny
		case 'S':
			x1, y1 := cx, cy
			if lastCmd == 'C' || lastCmd == 'c' || lastCmd == 'S' || lastCmd == 's' {
				x1, y1 = 2*cx-lcx, 2*cy-lcy
			}
			x2, y2 := off(args[0], args[1])
			nx, ny := off(args[2], args[3])
			p.CubeBezier(fpt(x1, y1), fpt(x2, y2), fpt(nx, ny))
			lcx, lcy = x2, y2
			cx, cy = nx, ny
		case 'Q':
			x1, y1 := off(args[0], args[1])
			nx, ny := off(args[2], args[3])
			p.QuadBezier(fpt(x1, y1), fpt(nx, ny))
			lcx, lcy = x1, y1
			cx, cy = nx, ny
		case 'T':
			x1, y1 := cx, cy
			if lastCmd == 'Q' || lastCmd == 'q' || lastCmd == 'T' || lastCmd == 't' {
				x1, y1 = 2*cx-lcx, 2*cy-lcy
			}
			nx, ny := off(args[0], args[1])
			p.QuadBezier(fpt(x1, y1), fpt(nx, ny))
			lcx, lcy = x1, y1
			cx, cy = nx, ny
		case 'A':
			rx, ry := args[0], args[1]
			rot := args[2]
			largeArc := args[3]
			sweep := args[4]
			nx, ny := off(args[5], args[6])
			addArc(&p, cx, cy, rx, ry, rot, largeArc, sweep, nx, ny)
			cx, cy = nx, ny
		case 'Z':
			p.Stop(true)
			cx, cy = sx, sy
		}

		// 记录本指令（反射判断用）
		lastCmd = cmd
	}

	return p
}

func isCmd(f string) bool {
	if len(f) != 1 {
		return false
	}
	c := f[0]
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func argCount(cmd byte) int {
	switch cmd {
	case 'M', 'L', 'T':
		return 2
	case 'H', 'V':
		return 1
	case 'C':
		return 6
	case 'S', 'Q':
		return 4
	case 'A':
		return 7
	case 'Z':
		return 0
	}
	return 0
}

// takeArgs 从 fields 取 n 个数字，推进 i。
func takeArgs(fields []string, i *int, n int) []float64 {
	args := make([]float64, 0, n)
	for len(args) < n && *i < len(fields) {
		f := fields[*i]
		if isCmd(f) {
			break
		}
		if v, ok := parseFloat(f); ok {
			args = append(args, v)
		}
		*i++
	}
	return args
}

// tokenizePath 把 path `d` 字符串切分成指令字母和数字 token。
// 例如 "M0 0h512" → ["M","0","0","h","512"]。
func tokenizePath(d string) []string {
	var tokens []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	for _, r := range d {
		switch {
		case r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'):
			// 指令字母：先结束当前数字 token，再作为独立 token。
			flush()
			cur.WriteRune(r)
			flush()
		case r == '-':
			// 负号：如果 cur 里有数字，先 flush（区分 "1-2" 这种情况）
			if cur.Len() > 0 {
				flush()
			}
			cur.WriteRune(r)
		case r == '.':
			// 若当前数字已含小数点，说明是新数字（如 ".9.4" → ".9", ".4"）
			if strings.ContainsRune(cur.String(), '.') {
				flush()
			}
			cur.WriteRune(r)
		case r >= '0' && r <= '9':
			cur.WriteRune(r)
		case r == 'e' || r == 'E':
			cur.WriteRune(r)
		}
	}
	flush()
	return tokens
}

// addArc 把一段 SVG 椭圆弧（endpoint 参数化）追加为三次贝塞尔段。
func addArc(p *rasterx.Path, x0, y0, rx, ry, rot, largeArc, sweep, x1, y1 float64) {
	if rx <= 0 || ry <= 0 {
		p.Line(fpt(x1, y1))
		return
	}
	rx = math.Abs(rx)
	ry = math.Abs(ry)
	phi := rot * math.Pi / 180

	// F.6.5：endpoint → center 参数化
	dx := (x0 - x1) / 2
	dy := (y0 - y1) / 2
	cosp := math.Cos(phi)
	sinp := math.Sin(phi)
	x1p := cosp*dx + sinp*dy
	y1p := -sinp*dx + cosp*dy

	// 半径过小则按比例放大
	lam := x1p*x1p/(rx*rx) + y1p*y1p/(ry*ry)
	if lam > 1 {
		s := math.Sqrt(lam)
		rx *= s
		ry *= s
	}

	num := rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p
	den := rx*rx*y1p*y1p + ry*ry*x1p*x1p
	var coef float64
	if den != 0 {
		coef = math.Sqrt(math.Max(0, num/den))
	}
	if largeArc == sweep {
		coef = -coef
	}
	cxp := coef * (rx * y1p / ry)
	cyp := coef * (-ry * x1p / rx)

	cx := cosp*cxp - sinp*cyp + (x0+x1)/2
	cy := sinp*cxp + cosp*cyp + (y0+y1)/2

	// 起始角与角度差
	theta1 := vectorAngle(1, 0, (x1p-cxp)/rx, (y1p-cyp)/ry)
	dtheta := vectorAngle((x1p-cxp)/rx, (y1p-cyp)/ry, (-x1p-cxp)/rx, (-y1p-cyp)/ry)
	if sweep == 0 && dtheta > 0 {
		dtheta -= 2 * math.Pi
	} else if sweep == 1 && dtheta < 0 {
		dtheta += 2 * math.Pi
	}

	// 分割成 ≤90° 的段
	n := int(math.Ceil(math.Abs(dtheta) / (math.Pi / 2)))
	if n < 1 {
		n = 1
	}
	for i := 0; i < n; i++ {
		t1 := theta1 + dtheta*float64(i)/float64(n)
		t2 := theta1 + dtheta*float64(i+1)/float64(n)
		addArcSegment(p, cx, cy, rx, ry, phi, t1, t2)
	}
}

// addArcSegment 把一个 ≤90° 的椭圆弧段转成一条三次贝塞尔。
func addArcSegment(p *rasterx.Path, cx, cy, rx, ry, phi, t0, t1 float64) {
	dt := t1 - t0
	alpha := 4.0 / 3.0 * math.Tan(dt/4)

	// 端点
	p0x, p0y := ellipsePoint(cx, cy, rx, ry, phi, t0)
	p3x, p3y := ellipsePoint(cx, cy, rx, ry, phi, t1)
	// 导数
	d0x, d0y := ellipseDeriv(cx, cy, rx, ry, phi, t0)
	d3x, d3y := ellipseDeriv(cx, cy, rx, ry, phi, t1)

	p1x := p0x + alpha*d0x
	p1y := p0y + alpha*d0y
	p2x := p3x - alpha*d3x
	p2y := p3y - alpha*d3y

	p.CubeBezier(fpt(p1x, p1y), fpt(p2x, p2y), fpt(p3x, p3y))
}

func ellipsePoint(cx, cy, rx, ry, phi, t float64) (float64, float64) {
	cosp := math.Cos(phi)
	sinp := math.Sin(phi)
	ex := rx * math.Cos(t)
	ey := ry * math.Sin(t)
	return cx + ex*cosp - ey*sinp, cy + ex*sinp + ey*cosp
}

func ellipseDeriv(cx, cy, rx, ry, phi, t float64) (float64, float64) {
	cosp := math.Cos(phi)
	sinp := math.Sin(phi)
	ex := -rx * math.Sin(t)
	ey := ry * math.Cos(t)
	return ex*cosp - ey*sinp, ex*sinp + ey*cosp
}

// vectorAngle 返回向量 (ux,uy) 到 (vx,vy) 的有符号夹角。
func vectorAngle(ux, uy, vx, vy float64) float64 {
	dot := ux*vx + uy*vy
	lenU := math.Hypot(ux, uy)
	lenV := math.Hypot(vx, vy)
	if lenU == 0 || lenV == 0 {
		return 0
	}
	c := dot / (lenU * lenV)
	c = math.Max(-1, math.Min(1, c))
	angle := math.Acos(c)
	cross := ux*vy - uy*vx
	if cross < 0 {
		angle = -angle
	}
	return angle
}
