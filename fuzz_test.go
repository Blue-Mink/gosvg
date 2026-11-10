package gosvg

import "testing"

// 本文件的 fuzz 目标只有一个不变量：对任意（含畸形）输入，各解析/渲染
// 入口绝不 panic —— 要么成功、要么返回 error/零值、要么静默降级。
// 正常 `go test` 只跑种子；`go test -fuzz=FuzzParse -fuzztime=30s` 才真正模糊测试。

func FuzzParse(f *testing.F) {
	seeds := []string{
		`<svg viewBox="0 0 100 100"><rect width="100" height="100" fill="red"/></svg>`,
		`<svg><path d="M0 0 L10 10 C20 20 30 30 40 40 Z"/></svg>`,
		`<svg><text x="0" y="0" font-size=".9.4">x</text></svg>`, // 紧凑小数踩坑
		`<svg><path d="M0 0 L1e-5 10"/></svg>`,                   // 科学计数法踩坑
		`<svg><circle cx="50" cy="50" r="40" fill="none" stroke="#fff"/></svg>`,
		`<svg><defs><linearGradient id="g"><stop offset="0" stop-color="red"/></linearGradient></defs><rect fill="url(#g)" width="100" height="100"/></svg>`,
		`<svg><style>.a{fill:red}</style><rect class="a" width="10" height="10"/></svg>`,
		`<svg><clipPath id="c"><circle r="10"/></clipPath><rect clip-path="url(#c)" width="50" height="50"/></svg>`,
		`<svg><marker id="m"><path d="M0 0L10 5L0 10z"/></marker><line x1="0" y1="0" x2="50" y2="50" marker-end="url(#m)"/></svg>`,
		`<svg viewBox="0 0 100 100" preserveAspectRatio="none"><rect width="100" height="200"/></svg>`,
		`<svg><image href="x" x="0" y="0" width="10" height="10"/></svg>`,
		`<svg><tspan>A<tspan>B</tspan></tspan></svg>`,
		`<svg><mask id="m"><rect width="10" height="10" fill="white"/></mask><rect mask="url(#m)"/></svg>`,
		`<svg><use href="#x"/></svg>`,
		`not xml at all`,
		`<svg><unclosed`,
		`<svg><rect width="abc" height="1e-5" rx=".9.4"/></svg>`,
		``,
		`<svg><path d="M0 0 A 10 10 0 1 1 20 20"/></svg>`,
		`<svg><ellipse cx="0" cy="0" rx="10" ry="10"/></svg>`,
		`<svg><polygon points="0,0 10,0 10,10"/></svg>`,
		`<svg><polyline points="0,0 10,0 10,10"/></svg>`,
		`<svg><line x1="0" y1="0" x2="10" y2="10" stroke-dasharray="5 5"/></svg>`,
		`<svg><text font-family="Arial, sans-serif" font-weight="bold">hi</text></svg>`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		img, err := Render(data, 64)
		_ = img
		_ = err
		doc, perr := Parse(data)
		if doc != nil {
			_, _ = doc.Render(32, 32)
		}
		_ = perr
	})
}

func FuzzPathData(f *testing.F) {
	for _, d := range []string{
		"M0 0 L10 10",
		"M0 0 L1e-5 10",
		"M0 0 C1 2 3 4 5 6 S7 8 9 10",
		"M0 0 Q1 2 3 4 T5 6",
		"M0 0 A10 10 0 0 1 20 20 Z",
		"M0 0 A10 10 0 1 0 20 20",
		"m.9.4 1,2",
		"M0 0",
		"",
		"garbage path data",
		"M0 0 L10 10 1e-5",
		"M 0 0 h 10 v 10 H 0 V 0 Z",
		"M0,0l10,10z",
		"M0 0 L10 10 M20 20 L30 30",
	} {
		f.Add(d)
	}
	f.Fuzz(func(t *testing.T, d string) {
		_ = parsePathData(d)
	})
}

func FuzzColor(f *testing.F) {
	for _, c := range []string{
		"red", "#fff", "#ff0000", "#ff000080", "#f00", "#ffff",
		"rgb(255,0,0)", "rgb(255 0 0)", "rgb(255 0 0 / 0.5)",
		"rgba(255,0,0,.5)", "hsl(120 50% 50%)", "hsla(120,50%,50%,.5)",
		"hsl(120,50%,50%)", "none", "currentColor", "url(#g)",
		"not-a-color", "", "#", "rgb()", "hsl()",
		"hsl(999 999% 999%)", "#.9.4", "rgb(1e-5,0,0)", "rgba(0,0,0,1e-5)",
	} {
		f.Add(c)
	}
	f.Fuzz(func(t *testing.T, c string) {
		_, _ = parseColor(c)
		_, _ = parseColorFunc(c)
		_, _ = parseColorComp(c)
	})
}

func FuzzStyle(f *testing.F) {
	for _, s := range []string{
		"fill:red;stroke:blue",
		"fill: #ff0000 ; stroke-width: 2px",
		"fill:red;fill:blue;;;",
		"invalid",
		"",
		"fill:url(#g)",
		"stroke-dasharray:5 5;stroke-linejoin:round",
		"fill:rgb(1e-5,0,0)",
		// CSS 字符串里的括号 / 分号不是语法字符：css.go 引号感知扫描的回归种子。
		`content:"{";fill:red`,
		`content:"}";fill:blue`,
		`content:"a;b";fill:green`,
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = parseStyle(s)
		_ = parseCSS("rect{" + s + "}")
	})
}

func FuzzViewBoxTransform(f *testing.F) {
	for _, s := range []string{
		"0 0 100 100",
		"0 0 100",
		"",
		"a b c d",
		"0 0 1e-5 100",
		"translate(10 20) scale(2) rotate(45)",
		"matrix(1 0 0 1 0 0)",
		"garbage",
		"",
		"rotate(45 10 20)",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = parseViewBox(s)
		_ = parseTransform(s)
	})
}

func FuzzRender(f *testing.F) {
	for _, s := range []string{
		`<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="red"/></svg>`,
		`<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="red" display="none"/></svg>`,
		`<svg viewBox="0 0 10 10"><defs><mask id="m"><rect width="10" height="10" fill="white"/></mask></defs><rect width="10" height="10" fill="red" mask="url(#m)"/></svg>`,
		`<svg viewBox="0 0 10 10"><defs><clipPath id="c"><circle cx="5" cy="5" r="4"/></clipPath></defs><rect width="10" height="10" fill="red" clip-path="url(#c)"/></svg>`,
		`<svg viewBox="0 0 10 10"><defs><linearGradient id="g"><stop offset="0" stop-color="red"/></linearGradient></defs><rect width="10" height="10" fill="url(#g)"/></svg>`,
		`<svg viewBox="0 0 10 10"><path d="M0 0 A1 1 0 1 1 2 2 Z" stroke="#000" stroke-width="2" stroke-dasharray="2 1"/></svg>`,
		`<svg viewBox="0 0 10 10"><defs><marker id="m" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="4" markerHeight="4"><path d="M0 0L10 5L0 10z"/></marker></defs><line x1="0" y1="0" x2="10" y2="10" stroke="#000" marker-end="url(#m)"/></svg>`,
		`<svg viewBox="0 0 10 10"><text x="1" y="5" font-size="4">hi</text></svg>`,
		`<svg><image href="data:image/png;base64,QUFBQQ=="/></svg>`,
		`<svg><style>.a{display:none}</style><g class="a"><rect width="4" height="4"/></g></svg>`,
		`<svg><use href="#a"/><use id="a" href="#use1"/></svg>`,
		``,
		`not xml`,
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Render(data, 32)
		doc, err := Parse(data)
		if err == nil && doc != nil {
			_, _ = doc.Render(16, 16)
		}
	})
}
