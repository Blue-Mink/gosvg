package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Blue-Mink/gosvg/internal/cmdtestutil"
)

func TestCliMainCorpus(t *testing.T) {
	// 注意：CaptureStdout 会临时替换全局 os.Stdout，
	// 本测试文件顶层不得 t.Parallel()，否则 -race 下会与其他捕获测试互踩。
	in := cmdtestutil.Dir(t, "corpus", cmdtestutil.GoodSVG)

	t.Run("text output", func(t *testing.T) {
		out := cmdtestutil.CaptureStdout(t, func() {
			if code := cliMain([]string{"gosvg-corpus", "-size", "16", in}); code != 0 {
				t.Errorf("cliMain = %d, want 0", code)
			}
		})
		if !strings.Contains(out, "SVG 文件数: 1") || !strings.Contains(out, "渲染出内容: 1") {
			t.Errorf("文本汇总缺关键字段: %q", out)
		}
	})

	t.Run("json output", func(t *testing.T) {
		out := cmdtestutil.CaptureStdout(t, func() {
			if code := cliMain([]string{"gosvg-corpus", "-json", in}); code != 0 {
				t.Errorf("cliMain = %d, want 0", code)
			}
		})
		var rep Report
		if err := json.Unmarshal([]byte(out), &rep); err != nil {
			t.Fatalf("-json 输出不是合法 Report: %v / %q", err, out)
		}
		if rep.Total != 1 || rep.WithContent != 1 || rep.Size != 64 {
			t.Errorf("Report = %+v, want Total=1 WithContent=1 Size=64", rep)
		}
	})

	t.Run("no dirs", func(t *testing.T) {
		if code := cliMain([]string{"gosvg-corpus"}); code != 2 {
			t.Errorf("缺目录应退出 2，got %d", code)
		}
	})

	t.Run("unknown flag", func(t *testing.T) {
		if code := cliMain([]string{"gosvg-corpus", "-bogus", in}); code != 2 {
			t.Errorf("未知 flag 应退出 2，got %d", code)
		}
	})

	t.Run("help", func(t *testing.T) {
		if code := cliMain([]string{"gosvg-corpus", "-h"}); code != 0 {
			t.Errorf("-h 应退出 0，got %d", code)
		}
	})
}

// printText 的 0% 覆盖补齐：构造含失败/全透明/诊断/总数为 0 的 Report，
// 走遍所有条件分支，断言关键文案都出现（不锁死百分比小数）。
func TestPrintTextBranches(t *testing.T) {
	// 注意：CaptureStdout 会临时替换全局 os.Stdout，
	// 本测试文件顶层不得 t.Parallel()，否则 -race 下会与其他捕获测试互踩。
	rep := &Report{
		Dirs:         []string{"d1", "d2"},
		Size:         32,
		Total:        4,
		WithContent:  1,
		Blank:        1,
		ParseFailed:  1,
		RenderFailed: 1,
		FailedFiles:  []string{"bad.svg :: parse err"},
		BlankFiles:   []string{"empty.svg"},
		Diagnostics:  map[string]int{"unsupported element skipped: filter": 2, "other": 1},
	}
	out := cmdtestutil.CaptureStdout(t, func() { printText(rep) })
	for _, want := range []string{
		"语料目录: d1, d2", "SVG 文件数: 4", "渲染出内容: 1", "渲染全透明: 1",
		"Parse 失败: 1", "Render 失败: 1", "失败样例", "全透明样例", "unsupported element skipped: filter",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("printText 输出缺 %q\n完整输出:\n%s", want, out)
		}
	}
	// Total=0 时 pct 走 0.0% 分支，且不 panic。
	zero := &Report{Dirs: []string{"empty"}}
	if out := cmdtestutil.CaptureStdout(t, func() { printText(zero) }); !strings.Contains(out, "0.0%") {
		t.Errorf("Total=0 应输出 0.0%%，got %q", out)
	}
}
