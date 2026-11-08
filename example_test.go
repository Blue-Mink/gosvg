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
