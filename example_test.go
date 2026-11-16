package gosvg_test

import (
	"fmt"
	"strings"
	"testing"

	gosvg "github.com/Blue-Mink/gosvg"
)

func ExampleRender() {
	svg := []byte(`<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="#4f8ef7"/></svg>`)
	img, err := gosvg.Render(svg, 64)
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	fmt.Println(img.Bounds().Dx(), img.Bounds().Dy())
	// Output: 64 64
}

func ExampleParse() {
	doc, err := gosvg.Parse([]byte(`<svg viewBox="0 0 10 10"><rect width="10" height="10" fill="red"/></svg>`))
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	small, err := doc.Render(16, 16)
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	large, err := doc.Render(32, 32)
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	fmt.Println(small.Bounds().Dx(), large.Bounds().Dx())
	// Output: 16 32
}

func ExampleMeasureText() {
	fmt.Println(gosvg.MeasureText("gosvg", 12, "sans-serif") > 0)
	// Output: true
}

func BenchmarkRender(b *testing.B) {
	data := []byte(`<svg viewBox="0 0 24 24"><path fill="none" stroke="#000" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"/></svg>`)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := gosvg.Render(data, 512); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParse(b *testing.B) {
	data := []byte(`<svg viewBox="0 0 24 24"><path fill="none" stroke="#000" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"/></svg>`)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := gosvg.Parse(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderDocument(b *testing.B) {
	data := []byte(`<svg viewBox="0 0 24 24"><path fill="none" stroke="#000" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"/></svg>`)
	doc, err := gosvg.Parse(data)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := doc.Render(512, 512); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRenderMultiStroke 用 60 条独立 stroke path，体现 strokeFiller 复用的收益：
// 复用前每条 path 都新建 ScannerGV+Filler（各自分配 w*h 缓冲），复用后只分配一次。
func BenchmarkRenderMultiStroke(b *testing.B) {
	var sb strings.Builder
	sb.WriteString(`<svg viewBox="0 0 200 200">`)
	for i := 0; i < 60; i++ {
		fmt.Fprintf(&sb, `<path d="M%d 10 L%d 190" stroke="#000" stroke-width="2" fill="none"/>`, i*3, i*3+40)
	}
	sb.WriteString(`</svg>`)
	data := []byte(sb.String())
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := gosvg.Render(data, 512); err != nil {
			b.Fatal(err)
		}
	}
}

// buildMaskClipSVG 构造 n 个「同时带 clipPath 与 mask」的分组。
// 每个分组渲染时，clip 与 mask 各要分配一张离屏位图（mask 还要 luminance 合成），
// 是「大批量 mask/clip 渲染」离屏位图分配热点的基准素材。
func buildMaskClipSVG(n int) []byte {
	var sb strings.Builder
	sb.WriteString(`<svg viewBox="0 0 200 200">`)
	sb.WriteString(`<defs>`)
	sb.WriteString(`<clipPath id="c"><circle cx="100" cy="100" r="80"/></clipPath>`)
	sb.WriteString(`<mask id="m"><rect width="200" height="200" fill="white"/><circle cx="100" cy="100" r="40" fill="black"/></mask>`)
	sb.WriteString(`</defs>`)
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, `<g clip-path="url(#c)" mask="url(#m)"><rect x="%d" y="10" width="40" height="180" fill="#4f8ef7"/></g>`, i*3)
	}
	sb.WriteString(`</svg>`)
	return []byte(sb.String())
}

// BenchmarkRenderMaskClip 用 16 个 mask+clip 分组，量化离屏位图分配开销。
// 若日后实现离位图池化，本 benchmark 的 B/op 与 allocs/op 应显著下降；
// 基准样例（BenchmarkRender）不含 mask/clip，无法反映这项收益。
// 用 -benchtime 控制轮数（单次渲染约数百 ms）。
func BenchmarkRenderMaskClip(b *testing.B) {
	data := buildMaskClipSVG(16)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := gosvg.Render(data, 512); err != nil {
			b.Fatal(err)
		}
	}
}
