// Command render 把单个 SVG 渲染成 PNG / JPEG 图片。
//
// 用法：
//
//	gosvg-render [-size N] [-W N] [-H N] [-format png|jpeg] [-quality N]
//	             [-bg #RRGGBB] [-strict] <input.svg|-> [output.ext]
//	gosvg-render [-size N] [-strict] <input.svg>            # 写到 stdout
//
// flag 必须写在位置参数之前（Go flag 包的标准行为）。
// 兼容旧用法：gosvg-render in.svg out.png [size]（第三个位置参数为尺寸，
// 且未显式给 -size/-W/-H 时才生效）。
// 省略 <output> 时结果写到 stdout；<input> 为 "-" 时从 stdin 读取。
// -strict 在渲染期出现任何静默跳过事件（未识别元素、缺失引用等）时退出 3，
// 事件文本写到 stderr；诊断通道只读，不改变渲染结果。
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	gosvg "github.com/Blue-Mink/gosvg"
)

func main() {
	os.Exit(cliMain(os.Args, os.Stdin, os.Stdout, os.Stderr))
}

// cliMain 是 main 的可测编排，返回退出码：
//
//	0 成功 / 1 读写渲染编码失败 / 2 参数错误 / 3 -strict 发现跳过事件
//
// in/out/err 由调用方注入，测试可捕获 stdin/stdout/stderr 而无需碰全局 os.Stdout。
func cliMain(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gosvg-render", flag.ContinueOnError)
	fs.SetOutput(stderr)
	size := fs.Int("size", 512, "渲染尺寸（宽高相同；-W/-H 优先）")
	width := fs.Int("W", 0, "输出宽度（0 = 用 -size）")
	height := fs.Int("H", 0, "输出高度（0 = 用 -size）")
	format := fs.String("format", "", "输出格式 png / jpeg；空则按输出扩展名推断，无扩展名默认 png")
	quality := fs.Int("quality", 90, "JPEG 质量 1-100")
	bg := fs.String("bg", "", "底色 #RGB / #RRGGBB（默认透明）")
	strict := fs.Bool("strict", false, "渲染期出现跳过事件则退出 3")
	if err := fs.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0 // -h 由 FlagSet 自己打印用法，不算错误
		}
		return 2
	}
	rest := fs.Args()
	if len(rest) < 1 || len(rest) > 3 {
		fmt.Fprintln(stderr, "usage: gosvg-render [flags] <input.svg|-> [output.ext [size]]")
		return 2
	}

	// 显式 -size/-W/-H 优先于旧式第三个位置参数；两者不可同时出现。
	explicitSize := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "size" || f.Name == "W" || f.Name == "H" {
			explicitSize = true
		}
	})
	if len(rest) == 3 && explicitSize {
		fmt.Fprintln(stderr, "usage: -size/-W/-H 与位置参数 size 不可同时使用")
		return 2
	}

	w, h := *width, *height
	if w <= 0 {
		w = *size
	}
	if h <= 0 {
		h = *size
	}
	if len(rest) == 3 {
		// 旧用法 in.svg out.png N：parseSize 按完整 os.Args 形式读第 3 位。
		v := parseSize(append([]string{args[0]}, rest...), *size)
		w, h = v, v
	}
	if w <= 0 || h <= 0 {
		fmt.Fprintln(stderr, "render: -size/-W/-H 必须为正数")
		return 2
	}
	if *quality < 0 || *quality > 100 {
		fmt.Fprintln(stderr, "render: -quality 必须在 0-100")
		return 2
	}

	var bgColor color.Color
	if *bg != "" {
		c, ok := parseHexColor(*bg)
		if !ok {
			fmt.Fprintf(stderr, "render: 非法 -bg %q（支持 #RGB / #RRGGBB）\n", *bg)
			return 2
		}
		bgColor = c
	}

	inPath, outPath := rest[0], ""
	if len(rest) >= 2 {
		outPath = rest[1]
	}

	var data []byte
	var err error
	if inPath == "-" {
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(inPath)
	}
	if err != nil {
		fmt.Fprintln(stderr, "read:", err)
		return 1
	}

	// 显式 Parse + RenderWithOptions，等价于库侧 RenderTo(data, w, h)，
	// 但能拿到 LastRenderDiagnostics() 供 -strict 判定。渲染路径逐字一致。
	doc, err := gosvg.Parse(data)
	if err != nil {
		fmt.Fprintln(stderr, "render:", err)
		return 1
	}
	opts := gosvg.DefaultRenderOptions()
	opts.JPEGQuality = *quality
	opts.Background = bgColor
	img, err := doc.RenderWithOptions(nil, w, h, opts)
	if err != nil {
		fmt.Fprintln(stderr, "render:", err)
		return 1
	}
	if *strict {
		diags := doc.LastRenderDiagnostics()
		if len(diags) > 0 {
			for _, d := range diags {
				fmt.Fprintln(stderr, "diagnostic:", d)
			}
			return 3
		}
	}

	f := *format
	if f == "" {
		f = formatFromPath(outPath)
	}

	// 先编码到内存：编码失败时不要留下一个半截的输出文件。
	var buf bytes.Buffer
	if err := gosvg.EncodeImage(&buf, img, f, *quality, bgColor); err != nil {
		fmt.Fprintln(stderr, "encode:", err)
		return 1
	}
	if outPath == "" {
		if _, err := stdout.Write(buf.Bytes()); err != nil {
			fmt.Fprintln(stderr, "write:", err)
			return 1
		}
		return 0
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		fmt.Fprintln(stderr, "write:", err)
		return 1
	}
	return 0
}

// formatFromPath 按输出文件扩展名推断格式；无扩展名或未知扩展名一律 png。
func formatFromPath(p string) string {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".jpg", ".jpeg":
		return "jpeg"
	default:
		return "png"
	}
}

// parseHexColor 解析 #RGB / #RRGGBB（CLI 专用；库侧 parseColor 未导出，
// 这里不引入新依赖，也不复制其 CSS 命名色 / rgb() 解析）。
func parseHexColor(s string) (color.NRGBA, bool) {
	if len(s) < 2 || s[0] != '#' {
		return color.NRGBA{}, false
	}
	hex := s[1:]
	hexv := func(c byte) (uint8, bool) {
		switch {
		case c >= '0' && c <= '9':
			return c - '0', true
		case c >= 'a' && c <= 'f':
			return c - 'a' + 10, true
		case c >= 'A' && c <= 'F':
			return c - 'A' + 10, true
		}
		return 0, false
	}
	switch len(hex) {
	case 3:
		var v [3]uint8
		for i := 0; i < 3; i++ {
			n, ok := hexv(hex[i])
			if !ok {
				return color.NRGBA{}, false
			}
			v[i] = n * 0x11
		}
		return color.NRGBA{R: v[0], G: v[1], B: v[2], A: 0xff}, true
	case 6:
		var v [3]uint8
		for i := 0; i < 3; i++ {
			hi, ok1 := hexv(hex[2*i])
			lo, ok2 := hexv(hex[2*i+1])
			if !ok1 || !ok2 {
				return color.NRGBA{}, false
			}
			v[i] = hi<<4 | lo
		}
		return color.NRGBA{R: v[0], G: v[1], B: v[2], A: 0xff}, true
	}
	return color.NRGBA{}, false
}

// parseSize 从完整参数（os.Args 形态）的第 3 个位置参数取渲染尺寸；
// 非法或非正一律回退默认值。仅用于兼容旧用法 in.svg out.png N。
func parseSize(args []string, def int) int {
	if len(args) > 3 {
		if v, err := strconv.Atoi(args[3]); err == nil && v > 0 {
			return v
		}
	}
	return def
}
