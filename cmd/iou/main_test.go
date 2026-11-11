package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	gosvg "github.com/Blue-Mink/gosvg"
)

func TestIouOf(t *testing.T) {
	for _, tc := range []struct {
		name string
		a, b []bool
		want float64
	}{
		{"完全相同", []bool{true, true, false}, []bool{true, true, false}, 1},
		{"部分重叠", []bool{true, true, false, false}, []bool{false, true, true, false}, 1.0 / 3.0},
		{"完全不相交", []bool{true, true}, []bool{false, false}, 0},
		{"两者全空视为一致", []bool{false, false}, []bool{false, false}, 1},
		{"一空一非空", []bool{true, false}, []bool{false, false}, 0},
		{"长度不等", []bool{true}, []bool{true, true}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := iouOf(tc.a, tc.b)
			if diff := got - tc.want; diff > 1e-9 || diff < -1e-9 {
				t.Fatalf("iouOf = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMaskOf(t *testing.T) {
	alpha := []uint8{0, 1, 8, 9, 255}
	m0 := maskOf(alpha, 0)
	if !m0[1] || !m0[2] || m0[0] {
		t.Fatalf("alpha>0 阈值错: %v", m0)
	}
	m8 := maskOf(alpha, 8)
	if m8[2] || !m8[3] || !m8[4] {
		t.Fatalf("alpha>8 阈值错（8 本身不应算不透明）: %v", m8)
	}
}

func TestResizeNearest(t *testing.T) {
	// 2×2 → 4×4，每维放大两倍，每个源像素应覆盖 2×2 目标块。
	src := []uint8{1, 2, 3, 4}
	got := resizeNearest(src, 2, 2, 4, 4)
	want := []uint8{1, 1, 2, 2, 1, 1, 2, 2, 3, 3, 4, 4, 3, 3, 4, 4}
	if len(got) != len(want) {
		t.Fatalf("长度 = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("resizeNearest[%d] = %d, want %d\n got=%v", i, got[i], want[i], got)
		}
	}
	// 恒等尺寸应原样返回（不复制也不缩放）。
	same := resizeNearest(src, 2, 2, 2, 2)
	for i := range src {
		if same[i] != src[i] {
			t.Fatalf("恒等尺寸被改动: %v", same)
		}
	}
}

func TestCountTrue(t *testing.T) {
	if n := countTrue([]bool{true, false, true, true}); n != 3 {
		t.Fatalf("countTrue = %d, want 3", n)
	}
}

// writeTestPNG 写一个纯色 RGBA PNG 供端到端测试用。
func writeTestPNG(t *testing.T, path string, w, h int, c color.NRGBA) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPNG(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.png")
	writeTestPNG(t, p, 4, 3, color.NRGBA{R: 255, A: 255})
	a, w, h, err := loadPNG(p)
	if err != nil {
		t.Fatal(err)
	}
	if w != 4 || h != 3 {
		t.Fatalf("尺寸 = %dx%d, want 4x3", w, h)
	}
	if len(a) != 12 {
		t.Fatalf("alpha 长度 = %d, want 12", len(a))
	}
	for i, v := range a {
		if v != 255 {
			t.Fatalf("alpha[%d] = %d, want 255", i, v)
		}
	}
}

func TestRenderSVG(t *testing.T) {
	dir := t.TempDir()
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4"><rect width="4" height="4" fill="black"/></svg>`
	if err := os.WriteFile(filepath.Join(dir, "a.svg"), []byte(svg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	r, err := renderSVG(dir, out, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 1 {
		t.Fatalf("渲染了 %d 个文件，应只渲染 .svg 一个", len(r))
	}
	g := r["a"]
	if g.w != 8 || g.h != 8 || len(g.alpha) != 64 {
		t.Fatalf("渲染尺寸错: %dx%d, alpha %d", g.w, g.h, len(g.alpha))
	}
	if _, err := os.Stat(filepath.Join(out, "a.png")); err != nil {
		t.Fatalf("未写出 PNG: %v", err)
	}
}

// TestEndToEndIdentical 端到端：同一张图当参照与渲染结果时 IoU 应为 1。
func TestEndToEndIdentical(t *testing.T) {
	dir := t.TempDir()
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4"><rect width="4" height="4" fill="black"/></svg>`
	if err := os.WriteFile(filepath.Join(dir, "s.svg"), []byte(svg), 0o644); err != nil {
		t.Fatal(err)
	}
	img, err := gosvg.Render([]byte(svg), 16)
	if err != nil {
		t.Fatal(err)
	}
	refDir := filepath.Join(dir, "ref")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(refDir, "s.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()

	r, err := renderSVG(dir, "", 16)
	if err != nil {
		t.Fatal(err)
	}
	refAlpha, refW, refH, err := loadPNG(filepath.Join(refDir, "s.png"))
	if err != nil {
		t.Fatal(err)
	}
	got := r["s"]
	gotAlpha := resizeNearest(got.alpha, got.w, got.h, refW, refH)
	v := iouOf(maskOf(refAlpha, 0), maskOf(gotAlpha, 0))
	if v != 1 {
		t.Fatalf("同图对比 IoU = %v, want 1（ref_pixels=%d got_pixels=%d）",
			v, countTrue(maskOf(refAlpha, 0)), countTrue(maskOf(gotAlpha, 0)))
	}
}
