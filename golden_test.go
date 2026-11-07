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

var updateGolden = flag.Bool("update", false, "rewrite testdata/golden.sha256 from current output")

// s60s 带 <text>：font_test.go 会调用 AddFontDir 注册系统字体，这是包级全局状态，
// 会让同一 SVG 在「单独跑 golden」与「跑完整 go test ./...」时渲染出不同字节。
// 字体隔离/Reset 属于 GOSVG_GAP_TODO.md 档 2 待决策项，在此之前不把带文字样例
// 纳入像素哈希基线；其余 6 个纯图形样例不受影响。
var goldenCases = []string{"blues", "cn", "country", "jazz", "lucide", "rock"}

const goldenFile = "testdata/golden.sha256"

// TestGoldenRegression 把各样例的渲染像素做 SHA256 比对，防止无意改变渲染结果。
// 基线是 gosvg 自身输出（不是 cairosvg）；有意改动后请用 -update 重新生成。
func TestGoldenRegression(t *testing.T) {
	got := make([]string, 0, len(goldenCases))
	for _, name := range goldenCases {
		data, err := os.ReadFile(filepath.Join("testdata", name+".svg"))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		img, err := Render(data, 256)
		if err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
		sum := sha256.Sum256(img.Pix)
		got = append(got, fmt.Sprintf("%s  %s", name, hex.EncodeToString(sum[:])))
	}
	if *updateGolden {
		if err := os.WriteFile(goldenFile, []byte(strings.Join(got, "\n")+"\n"), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated %s", goldenFile)
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
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("渲染输出与 golden 基线不一致\n--- got ---\n%s\n--- want ---\n%s\n"+
			"若是有意改动，请运行: go test -run TestGoldenRegression -update .",
			strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
