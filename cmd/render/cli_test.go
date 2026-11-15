package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Blue-Mink/gosvg/internal/cmdtestutil"
)

func TestCliMainRender(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	in := filepath.Join(dir, "in.svg")
	out := filepath.Join(dir, "out.png")
	if err := os.WriteFile(in, []byte(cmdtestutil.GoodSVG), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("ok", func(t *testing.T) {
		if code := cliMain([]string{"gosvg-render", in, out, "32"}); code != 0 {
			t.Fatalf("cliMain = %d, want 0", code)
		}
		f, err := os.Open(out)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		img, err := png.Decode(f)
		if err != nil {
			t.Fatalf("输出不是合法 PNG: %v", err)
		}
		if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 32 {
			t.Errorf("尺寸 = %v, want 32x32", img.Bounds())
		}
	})

	t.Run("too few args", func(t *testing.T) {
		if code := cliMain([]string{"gosvg-render"}); code != 2 {
			t.Errorf("参数不足应退出 2，got %d", code)
		}
	})

	t.Run("read failure", func(t *testing.T) {
		if code := cliMain([]string{"gosvg-render", filepath.Join(dir, "missing.svg"), out}); code != 1 {
			t.Errorf("读失败应退出 1，got %d", code)
		}
	})

	t.Run("render failure", func(t *testing.T) {
		bad := filepath.Join(dir, "bad.svg")
		if err := os.WriteFile(bad, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
		if code := cliMain([]string{"gosvg-render", bad, out}); code != 1 {
			t.Errorf("渲染失败应退出 1，got %d", code)
		}
	})

	t.Run("create failure", func(t *testing.T) {
		// 把输出路径指向一个目录，os.Create 必失败。
		if code := cliMain([]string{"gosvg-render", in, dir}); code != 1 {
			t.Errorf("建输出失败应退出 1，got %d", code)
		}
	})
}
