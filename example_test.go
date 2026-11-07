package gosvg_test

import (
	"fmt"
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
