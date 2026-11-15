package main

import (
	"encoding/json"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Blue-Mink/gosvg/internal/cmdtestutil"
)

// 造一套 ref/baseline/svg 目录：ref 是不透明方块，baseline 全透明，svg 渲染成不透明方块。
// 于是 compareBaseline 应判为 improved（新渲染 IoU=1 > 基线 IoU=0）。
func iouFixture(t *testing.T) (dir, svgDir, refDir, baseDir string) {
	t.Helper()
	dir = t.TempDir()
	svgDir = filepath.Join(dir, "svg")
	refDir = filepath.Join(dir, "ref")
	baseDir = filepath.Join(dir, "base")
	for _, d := range []string{svgDir, refDir, baseDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(svgDir, "s.svg"), []byte(cmdtestutil.GoodSVG), 0o644); err != nil {
		t.Fatal(err)
	}
	// ref：4x4 全不透明；baseline：4x4 全透明（alpha=0）。
	writeTestPNG(t, filepath.Join(refDir, "s.png"), 4, 4, color.NRGBA{R: 255, A: 255})
	writeTestPNG(t, filepath.Join(baseDir, "s.png"), 4, 4, color.NRGBA{})
	// 额外一个 baseline-only 样本 t.png（无对应 svg），走 got.alpha==nil 的基线分支。
	writeTestPNG(t, filepath.Join(refDir, "t.png"), 4, 4, color.NRGBA{R: 255, A: 255})
	writeTestPNG(t, filepath.Join(baseDir, "t.png"), 4, 4, color.NRGBA{R: 255, A: 255})
	return
}

func TestCliMainIou(t *testing.T) {
	// 注意：CaptureStdout 会临时替换全局 os.Stdout，
	// 本测试文件顶层不得 t.Parallel()，否则 -race 下会与其他捕获测试互踩。
	dir, svgDir, refDir, baseDir := iouFixture(t)

	t.Run("full flags with json", func(t *testing.T) {
		outDir := filepath.Join(dir, "out")
		jsonPath := filepath.Join(dir, "report.json")
		out := cmdtestutil.CaptureStdout(t, func() {
			code := cliMain([]string{
				"gosvg-iou", "-ref", refDir, "-svg", svgDir, "-out", outDir,
				"-baseline", baseDir, "-alpha", "8", "-size", "16", "-min", "0.9",
				"-json", jsonPath, "-quiet",
			})
			if code != 0 {
				t.Errorf("cliMain = %d, want 0", code)
			}
		})
		if !strings.Contains(out, "average IoU") || !strings.Contains(out, "vs baseline: improved=1") {
			t.Errorf("汇总输出缺关键字段: %q", out)
		}
		blob, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Fatalf("-json 未落盘: %v", err)
		}
		var rep report
		if err := json.Unmarshal(blob, &rep); err != nil {
			t.Fatalf("JSON 不是合法 report: %v", err)
		}
		if rep.Compared != 2 || rep.AlphaThreshold != 8 || rep.Size != 16 {
			t.Errorf("report = %+v, want Compared=2 Alpha=8 Size=16", rep)
		}
		if _, err := os.Stat(filepath.Join(outDir, "s.png")); err != nil {
			t.Errorf("-out 未落盘 s.png: %v", err)
		}
	})

	t.Run("baseline only no svg", func(t *testing.T) {
		out := cmdtestutil.CaptureStdout(t, func() {
			if code := cliMain([]string{"gosvg-iou", "-ref", refDir, "-baseline", baseDir}); code != 0 {
				t.Errorf("cliMain = %d, want 0", code)
			}
		})
		if !strings.Contains(out, "compared=2") {
			t.Errorf("仅基线模式应比较 2 个样本: %q", out)
		}
	})

	t.Run("missing ref", func(t *testing.T) {
		emptyRef := filepath.Join(dir, "emptyref")
		if err := os.MkdirAll(emptyRef, 0o755); err != nil {
			t.Fatal(err)
		}
		cmdtestutil.CaptureStdout(t, func() {
			if code := cliMain([]string{"gosvg-iou", "-ref", emptyRef, "-svg", svgDir}); code != 0 {
				t.Errorf("参照图全缺应正常退出 0，got %d", code)
			}
		})
	})

	t.Run("no ref flag", func(t *testing.T) {
		if code := cliMain([]string{"gosvg-iou", "-svg", svgDir}); code != 2 {
			t.Errorf("缺 -ref 应退出 2，got %d", code)
		}
	})

	t.Run("neither svg nor baseline", func(t *testing.T) {
		if code := cliMain([]string{"gosvg-iou", "-ref", refDir}); code != 2 {
			t.Errorf("缺 -svg/-baseline 应退出 2，got %d", code)
		}
	})

	t.Run("bad flag", func(t *testing.T) {
		if code := cliMain([]string{"gosvg-iou", "-bogus"}); code != 2 {
			t.Errorf("未知 flag 应退出 2，got %d", code)
		}
	})

	t.Run("help", func(t *testing.T) {
		if code := cliMain([]string{"gosvg-iou", "-h"}); code != 0 {
			t.Errorf("-h 应退出 0，got %d", code)
		}
	})

	t.Run("unreadable ref dir", func(t *testing.T) {
		// -baseline 指向不存在目录 → os.ReadDir 失败 → 退出 1。
		if code := cliMain([]string{"gosvg-iou", "-ref", refDir, "-baseline", filepath.Join(dir, "no-such")}); code != 1 {
			t.Errorf("基线目录不可读应退出 1，got %d", code)
		}
	})
}

// compareBaseline 的 0% 覆盖补齐：improved / regressed / unchanged 三分支分别用
// 不透明 / 全透明 / 与参照同形但 IoU 相等的构造断言，并覆盖 ref/base 缺失的 continue 分支。
func TestCompareBaselineBranches(t *testing.T) {
	// 注意：CaptureStdout 会临时替换全局 os.Stdout，
	// 本测试文件顶层不得 t.Parallel()，否则 -race 下会与其他捕获测试互踩。
	dir, _, refDir, baseDir := iouFixture(t)
	opaque := color.NRGBA{R: 255, A: 255}
	transparent := color.NRGBA{}
	writeTestPNG(t, filepath.Join(refDir, "imp.png"), 4, 4, opaque)
	writeTestPNG(t, filepath.Join(baseDir, "imp.png"), 4, 4, transparent)
	writeTestPNG(t, filepath.Join(refDir, "reg.png"), 4, 4, opaque)
	writeTestPNG(t, filepath.Join(baseDir, "reg.png"), 4, 4, opaque)
	writeTestPNG(t, filepath.Join(refDir, "same.png"), 4, 4, opaque)
	writeTestPNG(t, filepath.Join(baseDir, "same.png"), 4, 4, opaque)
	// noRef / noBase：分别缺 ref / 缺 baseline，触发两处 continue。
	writeTestPNG(t, filepath.Join(baseDir, "noRef.png"), 4, 4, opaque)
	writeTestPNG(t, filepath.Join(refDir, "noBase.png"), 4, 4, opaque)

	full := map[string]raster{
		"imp":    {alpha: make([]uint8, 16), w: 4, h: 4},
		"reg":    {alpha: make([]uint8, 16), w: 4, h: 4},
		"same":   {alpha: make([]uint8, 16), w: 4, h: 4},
		"noRef":  {alpha: make([]uint8, 16), w: 4, h: 4},
		"noBase": {alpha: make([]uint8, 16), w: 4, h: 4},
	}
	for _, k := range []string{"imp", "same", "noBase", "noRef"} {
		for i := range full[k].alpha {
			full[k].alpha[i] = 255
		}
	}
	// reg 的渲染全透明 → IoU 低于基线（基线不透明 IoU=1）。
	imp, reg, same := compareBaseline(baseDir, full, 0, refDir)
	if imp != 1 || reg != 1 || same != 1 {
		t.Errorf("compareBaseline = improved=%d regressed=%d unchanged=%d, want 1/1/1",
			imp, reg, same)
	}
	_ = dir
}
