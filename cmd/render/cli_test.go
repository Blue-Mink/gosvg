package main

import (
	"bytes"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Blue-Mink/gosvg/internal/cmdtestutil"
)

// run 是 cliMain 的测试包装：捕获 stdout/stderr，避免污染测试进程全局流。
func run(t *testing.T, in string, args ...string) (code int, out, errOut []byte) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code = cliMain(append([]string{"gosvg-render"}, args...),
		strings.NewReader(in), &stdout, &stderr)
	return code, stdout.Bytes(), stderr.Bytes()
}

func writeSVG(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCliMainRender(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	in := writeSVG(t, dir, "in.svg", cmdtestutil.GoodSVG)
	out := filepath.Join(dir, "out.png")

	t.Run("ok default size", func(t *testing.T) {
		if code, _, e := run(t, "", in, out); code != 0 {
			t.Fatalf("cliMain = %d, stderr=%s", code, e)
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
		if img.Bounds().Dx() != 512 || img.Bounds().Dy() != 512 {
			t.Errorf("默认尺寸 = %v, want 512x512", img.Bounds())
		}
	})

	t.Run("legacy positional size", func(t *testing.T) {
		p := filepath.Join(dir, "legacy.png")
		if code, _, e := run(t, "", in, p, "32"); code != 0 {
			t.Fatalf("cliMain = %d, stderr=%s", code, e)
		}
		f, _ := os.Open(p)
		defer f.Close()
		img, err := png.Decode(f)
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 32 {
			t.Errorf("尺寸 = %v, want 32x32", img.Bounds())
		}
	})

	t.Run("explicit -W/-H wins", func(t *testing.T) {
		p := filepath.Join(dir, "wh.png")
		if code, _, e := run(t, "", "-W", "40", "-H", "24", in, p); code != 0 {
			t.Fatalf("cliMain = %d, stderr=%s", code, e)
		}
		f, _ := os.Open(p)
		defer f.Close()
		img, err := png.Decode(f)
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() != 40 || img.Bounds().Dy() != 24 {
			t.Errorf("尺寸 = %v, want 40x24", img.Bounds())
		}
	})

	t.Run("legacy size conflicts with -size", func(t *testing.T) {
		if code, _, e := run(t, "", "-size", "16", in, out, "32"); code != 2 {
			t.Errorf("冲突应退出 2, got %d (stderr=%s)", code, e)
		}
	})

	t.Run("too few args", func(t *testing.T) {
		if code, _, _ := run(t, ""); code != 2 {
			t.Errorf("参数不足应退出 2，got %d", code)
		}
	})

	t.Run("too many args", func(t *testing.T) {
		if code, _, _ := run(t, "", in, out, "8", "extra"); code != 2 {
			t.Errorf("参数过多应退出 2，got %d", code)
		}
	})

	t.Run("unknown flag", func(t *testing.T) {
		if code, _, _ := run(t, "", "-nope", in, out); code != 2 {
			t.Errorf("未知 flag 应退出 2，got %d", code)
		}
	})

	t.Run("-h", func(t *testing.T) {
		if code, _, e := run(t, "", "-h"); code != 0 {
			t.Errorf("-h 应退出 0，got %d (stderr=%s)", code, e)
		}
	})

	t.Run("read failure", func(t *testing.T) {
		if code, _, _ := run(t, "", filepath.Join(dir, "missing.svg"), out); code != 1 {
			t.Errorf("读失败应退出 1，got %d", code)
		}
	})

	t.Run("render failure", func(t *testing.T) {
		bad := writeSVG(t, dir, "bad.svg", "")
		if code, _, _ := run(t, "", bad, out); code != 1 {
			t.Errorf("渲染失败应退出 1，got %d", code)
		}
	})

	t.Run("create failure", func(t *testing.T) {
		// 把输出路径指向一个目录，os.WriteFile 必失败。
		if code, _, _ := run(t, "", in, dir); code != 1 {
			t.Errorf("建输出失败应退出 1，got %d", code)
		}
	})
}

func TestCliMainStdinStdout(t *testing.T) {
	t.Parallel()
	t.Run("stdin to stdout", func(t *testing.T) {
		code, out, e := run(t, cmdtestutil.GoodSVG, "-size", "16", "-")
		if code != 0 {
			t.Fatalf("cliMain = %d, stderr=%s", code, e)
		}
		img, err := png.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("stdout 不是合法 PNG: %v", err)
		}
		if img.Bounds().Dx() != 16 {
			t.Errorf("尺寸 = %v, want 16x16", img.Bounds())
		}
	})

	t.Run("file to stdout", func(t *testing.T) {
		dir := t.TempDir()
		in := writeSVG(t, dir, "in.svg", cmdtestutil.GoodSVG)
		code, out, e := run(t, "", "-size", "8", in)
		if code != 0 {
			t.Fatalf("cliMain = %d, stderr=%s", code, e)
		}
		if _, err := png.Decode(bytes.NewReader(out)); err != nil {
			t.Fatalf("stdout 不是合法 PNG: %v", err)
		}
	})
}

func TestCliMainFormat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	in := writeSVG(t, dir, "in.svg", cmdtestutil.GoodSVG)

	t.Run("jpeg by extension", func(t *testing.T) {
		p := filepath.Join(dir, "out.jpg")
		if code, _, e := run(t, "", "-size", "8", in, p); code != 0 {
			t.Fatalf("cliMain = %d, stderr=%s", code, e)
		}
		f, _ := os.Open(p)
		defer f.Close()
		if _, err := jpeg.Decode(f); err != nil {
			t.Fatalf("输出不是合法 JPEG: %v", err)
		}
	})

	t.Run("jpeg by -format", func(t *testing.T) {
		p := filepath.Join(dir, "explicit.png") // 扩展名与 -format 相反
		if code, _, e := run(t, "", "-size", "8", "-format", "jpeg", "-quality", "75", in, p); code != 0 {
			t.Fatalf("cliMain = %d, stderr=%s", code, e)
		}
		f, _ := os.Open(p)
		defer f.Close()
		if _, err := jpeg.Decode(f); err != nil {
			t.Fatalf("-format jpeg 未生效: %v", err)
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		if code, _, _ := run(t, "", "-format", "gif", in, filepath.Join(dir, "x.gif")); code != 1 {
			t.Errorf("不支持格式应退出 1，got %d", code)
		}
	})

	t.Run("invalid quality", func(t *testing.T) {
		if code, _, _ := run(t, "", "-quality", "101", in, outPath(t, dir)); code != 2 {
			t.Errorf("非法 quality 应退出 2，got %d", code)
		}
	})
}

func outPath(t *testing.T, dir string) string {
	t.Helper()
	return filepath.Join(dir, "o.png")
}

func TestCliMainBackground(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// 只画中间一小块，四角保留底色，便于对像素断言。
	partial := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4"><rect x="1" y="1" width="2" height="2" fill="red"/></svg>`
	in := writeSVG(t, dir, "in.svg", partial)

	for _, tc := range []struct {
		name string
		bg   string
		want color.NRGBA
	}{
		{"short hex", "#0f8", color.NRGBA{R: 0, G: 0xff, B: 0x88, A: 0xff}},
		{"long hex", "#112233", color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(dir, tc.name+".png")
			if code, _, e := run(t, "", "-size", "8", "-bg", tc.bg, in, p); code != 0 {
				t.Fatalf("cliMain = %d, stderr=%s", code, e)
			}
			f, _ := os.Open(p)
			defer f.Close()
			img, err := png.Decode(f)
			if err != nil {
				t.Fatal(err)
			}
			// PNG 无损：左上角应恰为底色（中间被红色矩形覆盖）。
			got := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA)
			if got != tc.want {
				t.Errorf("左上像素 = %v, want %v", got, tc.want)
			}
		})
	}

	t.Run("invalid bg", func(t *testing.T) {
		if code, _, _ := run(t, "", "-bg", "#gggggg", in, outPath(t, dir)); code != 2 {
			t.Errorf("非法 -bg 应退出 2，got %d", code)
		}
	})
}

func TestCliMainStrict(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// <filter> 会触发 "unsupported element skipped: filter" 诊断。
	withSkip := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4"><filter id="f"/><rect width="4" height="4" fill="red"/></svg>`
	in := writeSVG(t, dir, "skip.svg", withSkip)
	out := filepath.Join(dir, "strict.png")

	t.Run("strict detects diagnostic", func(t *testing.T) {
		code, _, e := run(t, "", "-size", "8", "-strict", in, out)
		if code != 3 {
			t.Fatalf("-strict 应退出 3，got %d (stderr=%s)", code, e)
		}
		if !strings.Contains(string(e), "unsupported element skipped: filter") {
			t.Errorf("stderr 应含诊断事件，got %q", e)
		}
	})

	t.Run("strict clean svg passes", func(t *testing.T) {
		clean := writeSVG(t, dir, "clean.svg", cmdtestutil.GoodSVG)
		if code, _, e := run(t, "", "-size", "8", "-strict", clean, out); code != 0 {
			t.Fatalf("干净 SVG 应退出 0，got %d (stderr=%s)", code, e)
		}
	})

	t.Run("without strict diagnostics are ignored", func(t *testing.T) {
		if code, _, e := run(t, "", "-size", "8", in, out); code != 0 {
			t.Fatalf("非 -strict 应退出 0，got %d (stderr=%s)", code, e)
		}
	})
}

func TestFormatFromPath(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ p, want string }{
		{"a.png", "png"}, {"a.PNG", "png"}, {"a.jpg", "jpeg"},
		{"a.jpeg", "jpeg"}, {"a.JPEG", "jpeg"}, {"a", "png"}, {"a.gif", "png"},
	} {
		if got := formatFromPath(tc.p); got != tc.want {
			t.Errorf("formatFromPath(%q) = %q, want %q", tc.p, got, tc.want)
		}
	}
}

func TestParseHexColor(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		s    string
		ok   bool
		want color.NRGBA
	}{
		{"#f00", true, color.NRGBA{R: 255, G: 0, B: 0, A: 255}},
		{"#112233", true, color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 255}},
		{"#0F8", true, color.NRGBA{R: 0, G: 0xff, B: 0x88, A: 255}},
		{"", false, color.NRGBA{}},
		{"f00", false, color.NRGBA{}},
		{"#gg0000", false, color.NRGBA{}},
		{"#1234", false, color.NRGBA{}},
		{"#11223", false, color.NRGBA{}},
	} {
		got, ok := parseHexColor(tc.s)
		if ok != tc.ok || got != tc.want {
			t.Errorf("parseHexColor(%q) = (%v,%v), want (%v,%v)", tc.s, got, ok, tc.want, tc.ok)
		}
	}
}
