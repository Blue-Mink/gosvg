package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Blue-Mink/gosvg/internal/cmdtestutil"
)

// cliMain 的四条退出分支：全部成功 / 有渲染失败 / 无 .svg / 目录不可读。
// 逐个用独立临时目录，不依赖全局 flag 状态。
func TestCliMainExitCodes(t *testing.T) {
	// 注意：CaptureStdout 会临时替换全局 os.Stdout，
	// 本测试文件顶层不得 t.Parallel()，否则 -race 下会与其他捕获测试互踩。
	base := t.TempDir()

	t.Run("all ok", func(t *testing.T) {
		in := cmdtestutil.Dir(t, "ok", cmdtestutil.GoodSVG)
		out := filepath.Join(in, "out")
		outText := cmdtestutil.CaptureStdout(t, func() {
			if code := cliMain([]string{"gosvg-verify", in, out}); code != 0 {
				t.Errorf("cliMain = %d, want 0", code)
			}
		})
		if !strings.Contains(outText, "1/1 ok") {
			t.Errorf("汇总输出应含 1/1 ok，got %q", outText)
		}
		if _, err := os.Stat(filepath.Join(out, "a.png")); err != nil {
			t.Errorf("应写出 a.png: %v", err)
		}
	})

	t.Run("render failure", func(t *testing.T) {
		in := filepath.Join(base, "fail")
		if err := os.MkdirAll(in, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(in, "bad.svg"), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
		cmdtestutil.CaptureStdout(t, func() {
			if code := cliMain([]string{"gosvg-verify", in}); code != 1 {
				t.Errorf("渲染失败应退出 1，got %d", code)
			}
		})
	})

	t.Run("no svg", func(t *testing.T) {
		empty := filepath.Join(base, "empty")
		if err := os.MkdirAll(empty, 0o755); err != nil {
			t.Fatal(err)
		}
		if code := cliMain([]string{"gosvg-verify", empty}); code != 1 {
			t.Errorf("空目录应退出 1，got %d", code)
		}
	})

	t.Run("unreadable dir", func(t *testing.T) {
		if code := cliMain([]string{"gosvg-verify", filepath.Join(base, "no-such")}); code != 1 {
			t.Errorf("不存在目录应退出 1，got %d", code)
		}
	})
}

// 不传 outDir（只渲染不落盘）与传 outDir 两条路径都要走到 writePNG 分支的覆盖补齐。
func TestCliMainWithoutOutDir(t *testing.T) {
	in := cmdtestutil.Dir(t, "noout", cmdtestutil.GoodSVG)
	outText := cmdtestutil.CaptureStdout(t, func() {
		if code := cliMain([]string{"gosvg-verify", in}); code != 0 {
			t.Errorf("cliMain = %d, want 0", code)
		}
	})
	if !strings.Contains(outText, "OK a.svg") {
		t.Errorf("输出应含 OK a.svg，got %q", outText)
	}
}
