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
	size := 512
	if len(os.Args) > 3 {
		if v, err := strconv.Atoi(os.Args[3]); err == nil && v > 0 {
			size = v
		}
	}
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
