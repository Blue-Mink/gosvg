// Command basic 演示 gosvg 的最小用法：渲染一段 SVG 并写成 PNG。
//
// 运行：go run ./examples/basic
package main

import (
	"image/png"
	"log"
	"os"

	gosvg "github.com/Blue-Mink/gosvg"
)

func main() {
	const svg = `<svg viewBox="0 0 24 24">
  <rect x="2" y="2" width="20" height="20" rx="4" fill="#f2f4f8"/>
  <circle cx="12" cy="12" r="7" fill="#4f8ef7"/>
  <path d="M8 12h8M12 8v8" stroke="#fff" stroke-width="1.5" fill="none"/>
</svg>`

	img, err := gosvg.Render([]byte(svg), 256)
	if err != nil {
		log.Fatal("render: ", err)
	}
	out, err := os.Create("out.png")
	if err != nil {
		log.Fatal("create: ", err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		log.Fatal("encode: ", err)
	}
	log.Println("wrote out.png (256x256)")
}
