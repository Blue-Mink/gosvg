package gosvg

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// 注意：-update 会跳过全部比对并用当前输出重写基线。仅在「有意改变渲染输出」
// 时使用；常规开发与 CI 绝不传该 flag。
var updateGolden = flag.Bool("update", false, "rewrite testdata/golden.sha256 from current output (intended output changes only)")

// goldenSkip 列出因外部依赖暂不纳入像素哈希的样例。此前 s60s 因 font_test.go 的
// AddFontDir 污染包级字体注册表而被排除；现在 font_test.go 用 t.Cleanup(resetFonts)
// 在每个测试后复位，带 <text> 的样例已可稳定纳入，故该表为空。
var goldenSkip = map[string]bool{}

const goldenFile = "testdata/golden.sha256"

// goldenCases 自动枚举 testdata 下的 SVG，避免新增样例漏进基线。
func goldenCases(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("testdata", "*.svg"))
	if err != nil {
		t.Fatalf("glob testdata: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("testdata 下没有 SVG，golden 基线为空（不应发生）")
	}
	sort.Strings(paths)
	names := make([]string, 0, len(paths))
	for _, p := range paths {
		name := strings.TrimSuffix(filepath.Base(p), ".svg")
		if !goldenSkip[name] {
			names = append(names, name)
		}
	}
	return names
}

// renderGolden 渲染单个样例并做健全性检查，返回 "名称  sha256" 基线行。
func renderGolden(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", name+".svg")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	img, err := Render(data, 256)
	if err != nil {
		t.Fatalf("render %s: %v", path, err)
	}
	if img.Bounds().Empty() {
		t.Fatalf("%s: 渲染结果为空图", path)
	}
	// 防止「渲染成全透明」也被计入基线（空 SVG 或渲染被错误跳过应显式失败）。
	hasAlpha := false
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0 {
			hasAlpha = true
			break
		}
	}
	if !hasAlpha {
		t.Fatalf("%s: 渲染结果全透明，不应纳入基线", path)
	}
	sum := sha256.Sum256(img.Pix)
	return fmt.Sprintf("%s  %s", name, hex.EncodeToString(sum[:]))
}

// TestGoldenRegression 把各样例的渲染像素做 SHA256 比对，防止无意改变渲染结果。
// 基线是 gosvg 自身输出（不是 cairosvg）；有意改动后才用 -update 重新生成。
func TestGoldenRegression(t *testing.T) {
	got := make([]string, 0, 8)
	for _, name := range goldenCases(t) {
		got = append(got, renderGolden(t, name))
	}
	if *updateGolden {
		if err := os.WriteFile(goldenFile, []byte(strings.Join(got, "\n")+"\n"), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated %s (%d cases)", goldenFile, len(got))
		return
	}
	f, err := os.Open(goldenFile)
	if err != nil {
		t.Fatalf("open %s: %v (run: go test -run TestGoldenRegression -update .)", goldenFile, err)
	}
	defer f.Close()
	var want []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			want = append(want, line)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(want) == 0 {
		t.Fatalf("%s 为空，golden 校验不可用", goldenFile)
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("渲染输出与 golden 基线不一致\n--- got ---\n%s\n--- want ---\n%s\n"+
			"若是有意改动，请运行: go test -run TestGoldenRegression -update .",
			strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
