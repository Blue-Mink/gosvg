package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	gosvg "github.com/Blue-Mink/gosvg"
)

// 用法：gosvg-verify [testdata_dir] [out_dir]
// 不给 out_dir 时只做渲染校验、不落盘；给定时把每个样例渲染成 PNG。
func main() {
	inDir := "testdata"
	outDir := ""
	if len(os.Args) > 1 {
		inDir = os.Args[1]
	}
	if len(os.Args) > 2 {
		outDir = os.Args[2]
	}
	if outDir != "" {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "mkdir:", err)
			os.Exit(1)
		}
	}
	entries, err := os.ReadDir(inDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read dir:", err)
		os.Exit(1)
	}
	failed, count := 0, 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".svg") {
			continue
		}
		count++
		name := strings.TrimSuffix(e.Name(), ".svg")
		data, err := os.ReadFile(filepath.Join(inDir, e.Name()))
		if err != nil {
			fmt.Printf("READ ERR   %-16s %v\n", e.Name(), err)
			failed++
			continue
		}
		img, err := gosvg.Render(data, 256)
		if err != nil {
			fmt.Printf("RENDER ERR %-16s %v\n", e.Name(), err)
			failed++
			continue
		}
		if outDir != "" {
			f, err := os.Create(filepath.Join(outDir, name+".png"))
			if err != nil {
				fmt.Printf("CREATE ERR %-16s %v\n", e.Name(), err)
				failed++
				continue
			}
			encErr := png.Encode(f, img)
			closeErr := f.Close()
			if encErr != nil || closeErr != nil {
				fmt.Printf("WRITE ERR  %-16s enc=%v close=%v\n", e.Name(), encErr, closeErr)
				failed++
				continue
			}
		}
		fmt.Printf("OK %-16s %dx%d\n", e.Name(), img.Bounds().Dx(), img.Bounds().Dy())
	}
	if count == 0 {
		fmt.Fprintln(os.Stderr, "no .svg found in", inDir)
		os.Exit(1)
	}
	fmt.Printf("%d/%d ok\n", count-failed, count)
	if failed > 0 {
		os.Exit(1)
	}
}
