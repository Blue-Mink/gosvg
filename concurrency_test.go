package gosvg

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// TestConcurrentRenderSameDocument 用同一份 Document 并发渲染不同尺寸，
// 配合 `go test -race` 验证：renderContext 每次新建、诊断字段有锁保护。
func TestConcurrentRenderSameDocument(t *testing.T) {
	doc, err := Parse([]byte(`<svg viewBox="0 0 10 10">
  <rect width="10" height="10" fill="red"/>
  <image href="missing.png" width="4" height="4"/>
</svg>`))
	if err != nil {
		t.Fatal(err)
	}
	const n = 16
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			size := 24 + i
			img, err := doc.Render(size, size)
			if err != nil {
				errs <- fmt.Errorf("goroutine %d: %w", i, err)
				return
			}
			if got := img.Bounds().Dx(); got != size {
				errs <- fmt.Errorf("goroutine %d: width = %d, want %d", i, got, size)
				return
			}
			_ = doc.Diagnostics()
			_ = doc.LastRenderDiagnostics()
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	joined := strings.Join(doc.Diagnostics(), "\n")
	if !strings.Contains(joined, "ImageDecoder not set") {
		t.Fatalf("missing image diagnostic after concurrent renders:\n%s", joined)
	}
	if c := strings.Count(joined, "ImageDecoder not set"); c != 1 {
		t.Fatalf("diagnostics should be deduplicated, got %d entries:\n%s", c, joined)
	}
}

// TestConcurrentParseAndRender 验证包级入口在多 goroutine 下可用。
func TestConcurrentParseAndRender(t *testing.T) {
	svg := []byte(`<svg viewBox="0 0 8 8"><circle cx="4" cy="4" r="3" fill="#4f8ef7"/></svg>`)
	const n = 8
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			img, err := Render(svg, 16+i)
			if err != nil {
				errs <- fmt.Errorf("goroutine %d: %w", i, err)
				return
			}
			if img.Bounds().Dx() != 16+i {
				errs <- fmt.Errorf("goroutine %d: wrong size %d", i, img.Bounds().Dx())
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}
