package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	gosvg "github.com/Blue-Mink/gosvg"
)

// parseArgs 从 CLI 参数取输入目录与可选输出目录。
func parseArgs(args []string) (inDir, outDir string) {
	inDir, outDir = "testdata", ""
	if len(args) > 1 {
		inDir = args[1]
	}
	if len(args) > 2 {
		outDir = args[2]
	}
	return
}

// run 渲染 inDir 下所有 .svg（256×256）；outDir 非空时同时写出 PNG。
// 返回成功数与失败数；目录不可读或无法创建 outDir 时返回 error。
func run(inDir, outDir string) (ok, failed int, err error) {
	if outDir != "" {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return 0, 0, err
		}
	}
	entries, err := os.ReadDir(inDir)
	if err != nil {
		return 0, 0, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".svg") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".svg")
		data, rerr := os.ReadFile(filepath.Join(inDir, e.Name()))
		if rerr != nil {
			fmt.Printf("READ ERR   %-16s %v\n", e.Name(), rerr)
			failed++
			continue
		}
		img, ierr := gosvg.Render(data, 256)
		if ierr != nil {
			fmt.Printf("RENDER ERR %-16s %v\n", e.Name(), ierr)
			failed++
			continue
		}
		if outDir != "" {
			if werr := writePNG(filepath.Join(outDir, name+".png"), img); werr != nil {
				fmt.Printf("WRITE ERR  %-16s %v\n", e.Name(), werr)
				failed++
				continue
			}
		}
		fmt.Printf("OK %-16s %dx%d\n", e.Name(), img.Bounds().Dx(), img.Bounds().Dy())
		ok++
	}
	return ok, failed, nil
}

// writePNG 编码并落盘；编码与关闭的错误都会被上报。
func writePNG(path string, img *image.NRGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	encErr := png.Encode(f, img)
	closeErr := f.Close()
	if encErr != nil {
		return encErr
	}
	return closeErr
}

// 用法：gosvg-verify [testdata_dir] [out_dir]
// 不给 out_dir 时只做渲染校验、不落盘；给定时把每个样例渲染成 PNG。
// 退出码：0 = 全部成功；1 = 有失败、目录不可读或没有 .svg。
func main() {
	inDir, outDir := parseArgs(os.Args)
	ok, failed, err := run(inDir, outDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if ok+failed == 0 {
		fmt.Fprintln(os.Stderr, "no .svg found in", inDir)
		os.Exit(1)
	}
	fmt.Printf("%d/%d ok\n", ok, ok+failed)
	if failed > 0 {
		os.Exit(1)
	}
}
