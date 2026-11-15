// Package cmdtestutil 为 cmd/* 各包的 CLI 编排测试提供共享的临时目录与 stdout 捕获工具。
// 只被 _test.go 引用，不参与任何生产二进制。
package cmdtestutil

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const GoodSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4"><rect width="4" height="4" fill="red"/></svg>`

// Dir 建一个含 1 个可渲染 SVG 的临时目录，返回目录路径。
func Dir(t *testing.T, name string, body string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.svg"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// CaptureStdout 把标准输出接到管道，执行 fn 后返回捕获到的文本。
// 用于 printText / 汇总输出这类「只写 stdout」的分支，不改变被测函数签名。
func CaptureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	_ = w.Close()
	blob, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	return strings.TrimSpace(string(blob))
}
