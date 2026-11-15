// Command iou 对账 gosvg 渲染结果与参照实现的像素 IoU。
//
// 用途：仓库外的对账脚本只能一次性运行，数字无法复现。此工具把对账口径固化进
// 仓库，使 CAIROSVG_IOU.md 与 CHANGELOG.md 里的平均 IoU、IoU<0.5 计数、
// 「N 提升 / M 退化」等结论可被任何人复算。
//
// 用法：
//
//	gosvg-iou -ref <参照图目录> [-svg <SVG 目录>] [-out <渲染输出目录>]
//	          [-alpha <阈值>] [-baseline <基线图目录>] [-json <输出文件>]
//	          [-size <边长>] [-min <只列出低于此值的文件>] [-quiet]
//
// 目录约定：参照图与渲染输出按**同名 stem**配对（foo.svg ↔ foo.png）。
//
// 口径：IoU 取 alpha 通道二值 mask 的交集 / 并集。阈值默认 0（alpha>0 即视为
// 不透明），与 CAIROSVG_IOU.md 正文使用的 alpha>8 口径不同 —— 用 -alpha 8 切换。
// 两套数字不可直接横比。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gosvg "github.com/Blue-Mink/gosvg"
)

// alphaChannel 提取图像的 alpha 通道；非灰度/真彩色一律按 RGBA 处理。
func alphaChannel(img image.Image) []uint8 {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := make([]uint8, 0, w*h)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			// RGBA() 返回 16 位预乘值，右移 8 位得到 8 位 alpha。
			out = append(out, uint8(a>>8))
		}
	}
	return out
}

// maskOf 按阈值把 alpha 通道二值化（alpha>threshold 视为不透明）。
func maskOf(alpha []uint8, threshold uint8) []bool {
	m := make([]bool, len(alpha))
	for i, a := range alpha {
		m[i] = a > threshold
	}
	return m
}

// iouOf 求两个等长二值 mask 的 IoU。两个 mask 全空视为 IoU=1（两者都“什么都没画”，
// 算作一致）；一空一非空为 0。
func iouOf(a, b []bool) float64 {
	if len(a) != len(b) {
		return 0
	}
	inter, union := 0, 0
	for i := range a {
		if a[i] || b[i] {
			union++
		}
		if a[i] && b[i] {
			inter++
		}
	}
	if union == 0 {
		return 1
	}
	return float64(inter) / float64(union)
}

// resizeNearest 最近邻缩放到目标边长。参照图与渲染图尺寸不一致时（参照图常为
// 256×256、渲染可能是其它尺寸），先归一到同一尺寸再比较。
func resizeNearest(alpha []uint8, srcW, srcH, dstW, dstH int) []uint8 {
	if srcW == dstW && srcH == dstH {
		return alpha
	}
	out := make([]uint8, dstW*dstH)
	for y := 0; y < dstH; y++ {
		sy := y * srcH / dstH
		if sy >= srcH {
			sy = srcH - 1
		}
		for x := 0; x < dstW; x++ {
			sx := x * srcW / dstW
			if sx >= srcW {
				sx = srcW - 1
			}
			out[y*dstW+x] = alpha[sy*srcW+sx]
		}
	}
	return out
}

// loadPNG 读 PNG 并返回其 alpha 通道与尺寸。
func loadPNG(path string) ([]uint8, int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, 0, 0, err
	}
	b := img.Bounds()
	return alphaChannel(img), b.Dx(), b.Dy(), nil
}

// raster 是一张位图的 alpha 通道与像素尺寸。
type raster struct {
	alpha []uint8
	w, h  int
}

// renderSVG 渲染目录下的所有 SVG，返回 stem → raster 的映射。
func renderSVG(svgDir, outDir string, size int) (map[string]raster, error) {
	out := make(map[string]raster)
	if outDir != "" {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return nil, err
		}
	}
	entries, err := os.ReadDir(svgDir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".svg") {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(svgDir, e.Name()))
		if rerr != nil {
			fmt.Fprintf(os.Stderr, "READ ERR %s: %v\n", e.Name(), rerr)
			continue
		}
		img, ierr := gosvg.Render(data, size)
		if ierr != nil {
			fmt.Fprintf(os.Stderr, "RENDER ERR %s: %v\n", e.Name(), ierr)
			continue
		}
		stem := strings.TrimSuffix(e.Name(), ".svg")
		b := img.Bounds()
		out[stem] = raster{alpha: alphaChannel(img), w: b.Dx(), h: b.Dy()}
		if outDir != "" {
			if err := writePNG(filepath.Join(outDir, stem+".png"), img); err != nil {
				fmt.Fprintf(os.Stderr, "WRITE ERR %s: %v\n", e.Name(), err)
			}
		}
	}
	return out, nil
}

// sample 表示单个文件的对比结果。
type sample struct {
	Name       string  `json:"name"`
	IoU        float64 `json:"iou"`
	RefPixels  int     `json:"ref_pixels"`
	GotPixels  int     `json:"got_pixels"`
	RefMissing bool    `json:"ref_missing,omitempty"`
}

// report 是整体对账结果。
type report struct {
	AlphaThreshold int      `json:"alpha_threshold"`
	Size           int      `json:"size"`
	Compared       int      `json:"compared"`
	Missing        int      `json:"missing"`
	AverageIoU     float64  `json:"average_iou"`
	BelowHalf      int      `json:"below_0.5"`
	BelowMin       int      `json:"below_min"`
	Samples        []sample `json:"samples,omitempty"`
}

func main() {
	os.Exit(cliMain(os.Args))
}

// cliMain 是 main 的可测编排：局部 FlagSet 解析参数、返回退出码，不直接终止进程；
// 逻辑与重构前的 main 逐分支等价（含 -h 退出 0、参数错误退出 2）。
func cliMain(args []string) int {
	fs := flag.NewFlagSet("gosvg-iou", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		refDir  = fs.String("ref", "", "参照图目录（必填），内含 PNG")
		svgDir  = fs.String("svg", "", "SVG 目录；为空时只与 -baseline 比")
		outDir  = fs.String("out", "", "渲染输出目录；为空则不落盘")
		alpha   = fs.Uint("alpha", 0, "alpha 阈值：alpha>阈值 视为不透明（0=alpha>0，8=报告口径）")
		baseDir = fs.String("baseline", "", "基线图目录；给定则额外算与基线的 IoU 差异")
		jsonOut = fs.String("json", "", "把完整结果写成 JSON 到该文件")
		size    = fs.Int("size", 256, "渲染边长")
		minVal  = fs.Float64("min", 0, "只列出 IoU 低于此值的文件（0=不限制）")
		quiet   = fs.Bool("quiet", false, "只输出汇总，不逐文件打印")
	)
	if err := fs.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	if *refDir == "" {
		fmt.Fprintln(os.Stderr, "usage: gosvg-iou -ref <参照图目录> [-svg <SVG 目录>] ...")
		return 2
	}
	if *svgDir == "" && *baseDir == "" {
		fmt.Fprintln(os.Stderr, "gosvg-iou: 至少需要 -svg 或 -baseline 之一")
		return 2
	}

	rep := report{AlphaThreshold: int(*alpha), Size: *size}

	// 渲染 SVG：-svg 给定时现场渲染；否则只走基线对比。
	var rendered map[string]raster
	if *svgDir != "" {
		if *outDir != "" {
			if err := os.MkdirAll(*outDir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "gosvg-iou: %v\n", err)
				return 1
			}
		}
		r, err := renderSVG(*svgDir, *outDir, *size)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gosvg-iou: %v\n", err)
			return 1
		}
		rendered = r
	}

	// 组装待比较的 stem 集合。
	stems := map[string]bool{}
	if *svgDir != "" {
		for s := range rendered {
			stems[s] = true
		}
	}
	if *baseDir != "" {
		entries, err := os.ReadDir(*baseDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gosvg-iou: %v\n", err)
			return 1
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".png") {
				continue
			}
			stems[strings.TrimSuffix(e.Name(), ".png")] = true
		}
	}
	names := make([]string, 0, len(stems))
	for s := range stems {
		names = append(names, s)
	}
	sort.Strings(names)

	threshold := uint8(*alpha)
	total := 0.0
	for _, stem := range names {
		refAlpha, refW, refH, rerr := loadPNG(filepath.Join(*refDir, stem+".png"))
		if rerr != nil {
			// 参照图缺失：跳过该样本（cairosvg 渲染失败的语料）。
			rep.Missing++
			continue
		}
		got := rendered[stem]
		if got.alpha == nil {
			// 没有渲染结果：试基线目录。
			if *baseDir == "" {
				rep.Missing++
				continue
			}
			ga, gw, gh, gerr := loadPNG(filepath.Join(*baseDir, stem+".png"))
			if gerr != nil {
				rep.Missing++
				continue
			}
			got = raster{alpha: resizeNearest(ga, gw, gh, refW, refH), w: refW, h: refH}
		} else {
			got = raster{alpha: resizeNearest(got.alpha, got.w, got.h, refW, refH), w: refW, h: refH}
		}
		gotAlpha := got.alpha
		v := iouOf(maskOf(refAlpha, threshold), maskOf(gotAlpha, threshold))
		s := sample{Name: stem, IoU: v}
		s.RefPixels = countTrue(maskOf(refAlpha, threshold))
		s.GotPixels = countTrue(maskOf(gotAlpha, threshold))
		rep.Samples = append(rep.Samples, s)
		total += v
		rep.Compared++
		if v < 0.5 {
			rep.BelowHalf++
		}
		if *minVal > 0 && v < *minVal {
			rep.BelowMin++
			if !*quiet {
				fmt.Printf("LOW  %-34s iou=%.4f ref=%d got=%d\n", stem, v, s.RefPixels, s.GotPixels)
			}
		}
	}

	if rep.Compared > 0 {
		rep.AverageIoU = total / float64(rep.Compared)
	}

	// 汇总输出。
	fmt.Printf("\nalpha>%d  size=%d  compared=%d  missing=%d\n", rep.AlphaThreshold, rep.Size, rep.Compared, rep.Missing)
	fmt.Printf("average IoU = %.6f\n", rep.AverageIoU)
	fmt.Printf("IoU < 0.5   = %d\n", rep.BelowHalf)

	// 与基线对比（若有）：统计提升 / 退化 / 不变。
	if *baseDir != "" && *svgDir != "" {
		imp, reg, same := compareBaseline(*baseDir, rendered, threshold, *refDir)
		fmt.Printf("vs baseline: improved=%d regressed=%d unchanged=%d\n", imp, reg, same)
	}

	if *jsonOut != "" {
		blob, err := json.MarshalIndent(rep, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "gosvg-iou: %v\n", err)
			return 1
		}
		if err := os.WriteFile(*jsonOut, blob, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "gosvg-iou: %v\n", err)
			return 1
		}
		fmt.Printf("JSON written to %s\n", *jsonOut)
	}
	return 0
}

// countTrue 数二值 mask 里 true 的个数（即不透明像素数）。
func countTrue(m []bool) int {
	n := 0
	for _, v := range m {
		if v {
			n++
		}
	}
	return n
}

// compareBaseline 以「渲染图 vs 参照图」的 IoU 变化方向统计提升 / 退化 / 不变：
// 高于基线 = 改善，低于 = 退化。
func compareBaseline(baseDir string, rendered map[string]raster, threshold uint8, refDir string) (imp, reg, same int) {
	for stem, got := range rendered {
		refAlpha, refW, refH, err := loadPNG(filepath.Join(refDir, stem+".png"))
		if err != nil {
			continue
		}
		baseAlpha, bw, bh, err := loadPNG(filepath.Join(baseDir, stem+".png"))
		if err != nil {
			continue
		}
		// 统一到参照图尺寸后比较各自的 IoU。
		newAlpha := resizeNearest(got.alpha, got.w, got.h, refW, refH)
		oldAlpha := resizeNearest(baseAlpha, bw, bh, refW, refH)
		refMask := maskOf(refAlpha, threshold)
		newIoU := iouOf(refMask, maskOf(newAlpha, threshold))
		oldIoU := iouOf(refMask, maskOf(oldAlpha, threshold))
		switch {
		case newIoU > oldIoU+0.0001:
			imp++
		case newIoU < oldIoU-0.0001:
			reg++
		default:
			same++
		}
	}
	return imp, reg, same
}

// writePNG 编码并落盘 PNG。
func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
