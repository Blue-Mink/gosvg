// Command options 演示 RenderOptions、context 取消与 Diagnostics 的用法。
//
// 运行：go run ./examples/options
package main

import (
	"context"
	"fmt"
	"image/color"

	gosvg "github.com/Blue-Mink/gosvg"
)

func main() {
	// 含一个超出子集的元素与一个未注入解码器的 <image>：两者都会被跳过并留下诊断。
	svg := []byte(`<svg viewBox="0 0 10 10">
  <filter id="f"/>
  <rect width="10" height="10" fill="#4f8ef7"/>
  <image href="missing.png" width="4" height="4"/>
</svg>`)

	opts := gosvg.RenderOptions{
		MaxDimension: 4096,
		Background:   color.NRGBA{R: 255, G: 255, B: 255, A: 255},
	}
	img, err := gosvg.RenderWithOptions(context.Background(), svg, 128, 128, opts)
	if err != nil {
		panic(err)
	}
	fmt.Println("size:", img.Bounds().Dx(), "x", img.Bounds().Dy())

	doc, err := gosvg.Parse(svg)
	if err != nil {
		panic(err)
	}
	if _, err := doc.Render(64, 64); err != nil {
		panic(err)
	}
	fmt.Println("diagnostics:")
	for _, d := range doc.Diagnostics() {
		fmt.Println("  -", d)
	}
}
