package gosvg

import (
	"testing"
	"time"
)

// runNoHang 用 goroutine + timeout 守护 fn，卡住即 FAIL。
func runNoHang(t *testing.T, name string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer func() { _ = recover() }()
		fn()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s 疑似死循环", name)
	}
}

// TestParsePathNoHang 回归：argCount==0 的指令（Z/z 或未知字母）后跟数字时，
// parsePathData 主循环的 `i--` 回退 + takeArgs(n=0) 不推进 i，曾死循环。
func TestParsePathNoHang(t *testing.T) {
	inputs := []string{
		"Z0",
		"Z 0",
		"z0",
		"M0 0 Z 5 5",
		"X0",
		"M0 0 Z",
		"M0 0 L1 1 Z1e5",
		"Z 1 2 3",
		"M0 0h512v512H0z", // 正常输入，不应误伤
	}
	for _, d := range inputs {
		d := d
		runNoHang(t, "parsePathData", func() { _ = parsePathData(d) })
	}
}

// TestRenderNoHang 回归：循环引用不应让 Render 无限递归（栈溢出）或死循环。
// marker 自引/互引是本轮修复的真 bug（renderMarker 缺深度限制）；
// use 靠 useDepth、clip/mask 靠 renderNodeBare 绕过 clip/mask 检查，本就安全，作为对照。
func TestRenderNoHang(t *testing.T) {
	svgs := []string{
		`<svg><defs><marker id="m"><path d="M0 0 L10 0" marker-start="url(#m)"/></marker></defs><path d="M0 0 L100 0" marker-start="url(#m)"/></svg>`,
		`<svg><defs><marker id="a"><path d="M0 0 L10 0" marker-start="url(#b)"/></marker><marker id="b"><path d="M0 0 L10 0" marker-start="url(#a)"/></marker></defs><path d="M0 0 L100 0" marker-start="url(#a)"/></svg>`,
		`<svg><defs><path id="a" d="M0 0 L10 0"/><use id="b" href="#a"/></defs><use href="#b"/></svg>`,
		`<svg><defs><clipPath id="c"><rect width="10" height="10" clip-path="url(#c)"/></clipPath></defs><rect width="50" height="50" clip-path="url(#c)"/></svg>`,
		`<svg><defs><mask id="m"><rect width="10" height="10" mask="url(#m)"/></mask></defs><rect width="50" height="50" mask="url(#m)"/></svg>`,
	}
	for _, svg := range svgs {
		svg := svg
		runNoHang(t, "Render", func() { _, _ = Render([]byte(svg), 64) })
	}
}
