// Command corpus 批量渲染目录下的所有 SVG，产出真实语料回归报告：
// 成功率、全透明清单、Parse/Render 失败清单与诊断事件分布。
//
// 用法：gosvg-corpus [-size N] [-json] <dir>...
//
// 与 testdata/golden.sha256 的区别：golden 是 testdata/*.svg 下固定样例的像素哈希
// （样例数由目录自动枚举，见 GOSVG_TODO.md 质量基线锚点；勿在此写死数量），
// 用于防无意改动；corpus 面向**外部真实语料**（系统图标库、node_modules 里的
// 图标集等），用于回答「这批真实 SVG 有多少能正确渲染、还缺什么能力」。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	gosvg "github.com/Blue-Mink/gosvg"
)

// Report 是一次语料扫描的汇总结果。
type Report struct {
	Dirs         []string       `json:"dirs"`
	Size         int            `json:"size"`
	Total        int            `json:"total"`
	WithContent  int            `json:"with_content"`
	Blank        int            `json:"blank"`
	ParseFailed  int            `json:"parse_failed"`
	RenderFailed int            `json:"render_failed"`
	ElapsedMs    int64          `json:"elapsed_ms"`
	BlankFiles   []string       `json:"blank_files,omitempty"`
	FailedFiles  []string       `json:"failed_files,omitempty"`
	Diagnostics  map[string]int `json:"diagnostics,omitempty"`
}

// collect 递归收集目录下所有 .svg，跳过 macOS AppleDouble（"._" 前缀）。
func collect(dirs []string) ([]string, error) {
	var files []string
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // 单个不可读条目不致命
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasPrefix(d.Name(), "._") {
				return nil
			}
			if strings.EqualFold(filepath.Ext(p), ".svg") {
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

// analyze 渲染每个 SVG 并汇总统计。
func analyze(files []string, size int) *Report {
	rep := &Report{Size: size, Diagnostics: map[string]int{}}
	start := time.Now()
	for _, f := range files {
		rep.Total++
		data, err := os.ReadFile(f)
		if err != nil {
			rep.ParseFailed++
			rep.addFailed(f, err)
			continue
		}
		doc, err := gosvg.Parse(data)
		if err != nil {
			rep.ParseFailed++
			rep.addFailed(f, err)
			continue
		}
		img, err := doc.Render(size, size)
		if err != nil {
			rep.RenderFailed++
			rep.addFailed(f, err)
			continue
		}
		n := 0
		for i := 3; i < len(img.Pix); i += 4 {
			if img.Pix[i] > 0 {
				n++
			}
		}
		if n == 0 {
			rep.Blank++
			if len(rep.BlankFiles) < 20 {
				rep.BlankFiles = append(rep.BlankFiles, filepath.Base(f))
			}
		} else {
			rep.WithContent++
		}
		for _, d := range doc.Diagnostics() {
			rep.Diagnostics[normalizeDiag(d)]++
		}
	}
	rep.ElapsedMs = time.Since(start).Milliseconds()
	return rep
}

func (r *Report) addFailed(f string, err error) {
	if len(r.FailedFiles) < 20 {
		r.FailedFiles = append(r.FailedFiles, filepath.Base(f)+" :: "+err.Error())
	}
}

// normalizeDiag 把带具体 id 的诊断归一化，便于跨语料聚合。
func normalizeDiag(d string) string {
	if strings.HasPrefix(d, "unsupported element skipped: ") {
		return d // 元素名本身就是分类信息，保留
	}
	if i := strings.Index(d, ": "); i > 0 {
		return d[:i+2] + "<id>"
	}
	return d
}

func main() {
	size := flag.Int("size", 64, "渲染边长（像素）")
	asJSON := flag.Bool("json", false, "以 JSON 输出报告")
	flag.Parse()
	dirs := flag.Args()
	if len(dirs) == 0 {
		fmt.Fprintln(os.Stderr, "usage: gosvg-corpus [-size N] [-json] <dir>...")
		os.Exit(2)
	}
	files, err := collect(dirs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "collect:", err)
		os.Exit(1)
	}
	rep := analyze(files, *size)
	rep.Dirs = dirs
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			fmt.Fprintln(os.Stderr, "encode:", err)
			os.Exit(1)
		}
		return
	}
	printText(rep)
}

func printText(r *Report) {
	pct := func(a int) string {
		if r.Total == 0 {
			return "0.0%"
		}
		return fmt.Sprintf("%.1f%%", float64(a)*100/float64(r.Total))
	}
	fmt.Printf("语料目录: %s\n", strings.Join(r.Dirs, ", "))
	fmt.Printf("SVG 文件数: %d   渲染尺寸: %dx%d   耗时: %dms\n", r.Total, r.Size, r.Size, r.ElapsedMs)
	fmt.Printf("渲染出内容: %d (%s)\n", r.WithContent, pct(r.WithContent))
	fmt.Printf("渲染全透明: %d (%s)\n", r.Blank, pct(r.Blank))
	fmt.Printf("Parse 失败: %d (%s)\n", r.ParseFailed, pct(r.ParseFailed))
	fmt.Printf("Render 失败: %d\n", r.RenderFailed)
	if len(r.FailedFiles) > 0 {
		fmt.Println("--- 失败样例(最多 20) ---")
		for _, f := range r.FailedFiles {
			fmt.Println("  ", f)
		}
	}
	if len(r.BlankFiles) > 0 {
		fmt.Println("--- 全透明样例(最多 20) ---")
		for _, f := range r.BlankFiles {
			fmt.Println("  ", f)
		}
	}
	type kv struct {
		k string
		v int
	}
	var ks []kv
	for k, v := range r.Diagnostics {
		ks = append(ks, kv{k, v})
	}
	sort.Slice(ks, func(i, j int) bool {
		if ks[i].v != ks[j].v {
			return ks[i].v > ks[j].v
		}
		return ks[i].k < ks[j].k
	})
	if len(ks) > 0 {
		fmt.Println("--- 诊断事件分布 ---")
		for i, e := range ks {
			if i >= 20 {
				fmt.Printf("  ... 其余 %d 类省略\n", len(ks)-20)
				break
			}
			fmt.Printf("  %5d  %s\n", e.v, e.k)
		}
	}
}
