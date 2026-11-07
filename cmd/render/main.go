package main

import (
	"fmt"
	"image/png"
	"os"
	"strconv"

	gosvg "github.com/Blue-Mink/gosvg"
)

// 用法：gosvg-render <input.svg> <output.png> [size]
func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: gosvg-render in.svg out.png [size]")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}
	size := parseSize(os.Args, 512)
	img, err := gosvg.Render(data, size)
	if err != nil {
		fmt.Fprintln(os.Stderr, "render:", err)
		os.Exit(1)
	}
	out, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}

// parseSize 从 CLI 参数取渲染尺寸；非法或非正一律回退默认值。
func parseSize(args []string, def int) int {
	if len(args) > 3 {
		if v, err := strconv.Atoi(args[3]); err == nil && v > 0 {
			return v
		}
	}
	return def
}
