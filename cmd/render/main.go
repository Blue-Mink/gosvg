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
	os.Exit(cliMain(os.Args))
}

// cliMain 是 main 的可测编排，返回退出码（0 成功 / 1 读写渲染失败 / 2 参数不足）。
func cliMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: gosvg-render in.svg out.png [size]")
		return 2
	}
	data, err := os.ReadFile(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		return 1
	}
	size := parseSize(args, 512)
	img, err := gosvg.Render(data, size)
	if err != nil {
		fmt.Fprintln(os.Stderr, "render:", err)
		return 1
	}
	out, err := os.Create(args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		return 1
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		return 1
	}
	return 0
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
