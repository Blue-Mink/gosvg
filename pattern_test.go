package gosvg

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"strings"
	"testing"
)

func TestPatternFillUserSpaceOnUse(t *testing.T) {
	svg := `<svg viewBox="0 0 10 10">
  <defs><pattern id="p" patternUnits="userSpaceOnUse" width="2" height="2">
    <rect width="1" height="1" fill="red"/>
  </pattern></defs>
  <rect width="10" height="10" fill="url(#p)"/>
</svg>`
	img, err := Render([]byte(svg), 20)
	if err != nil {
		t.Fatal(err)
	}
	n := nonTransparent(img)
	if n == 0 {
		t.Fatal("pattern fill produced nothing")
	}
	// 2×2 用户单位 tile 里只有 1×1 红块 → 约 1/4 覆盖（20×20=400）
	if n > 400/2 {
		t.Fatalf("pattern should tile sparsely, got %d/400", n)
	}
}

func TestPatternFillObjectBoundingBoxDefault(t *testing.T) {
	// patternUnits 默认 objectBoundingBox：width=0.5 → tile 为 bbox 的一半（5 用户单位）
	// patternContentUnits 默认 userSpaceOnUse：内容 5×5 用户单位正好填满 tile
	svg := `<svg viewBox="0 0 10 10">
  <defs><pattern id="p" width="0.5" height="0.5">
    <rect width="5" height="5" fill="blue"/>
  </pattern></defs>
  <rect width="10" height="10" fill="url(#p)"/>
</svg>`
	img, err := Render([]byte(svg), 16)
	if err != nil {
		t.Fatal(err)
	}
	if n := nonTransparent(img); n != 256 {
		t.Fatalf("objectBoundingBox pattern should cover fully, got %d/256", n)
	}
}

func TestPatternWithViewBox(t *testing.T) {
	svg := `<svg viewBox="0 0 10 10">
  <defs><pattern id="p" patternUnits="userSpaceOnUse" width="4" height="4" viewBox="0 0 100 100">
    <rect width="100" height="100" fill="green"/>
  </pattern></defs>
  <rect width="10" height="10" fill="url(#p)"/>
</svg>`
	img, err := Render([]byte(svg), 16)
	if err != nil {
		t.Fatal(err)
	}
	if n := nonTransparent(img); n != 256 {
		t.Fatalf("pattern viewBox should map content to tile, got %d/256", n)
	}
}

// 真实语料里的 Figma 导出模式：<pattern> + <use> + base64 <image>。
// 修复前这类 SVG 渲染为全透明（fnOS 面板的 lock-t_gZZsqx.svg）。
func TestPatternFigmaStyleWithUseAndImage(t *testing.T) {
	var buf bytes.Buffer
	px := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for i := 0; i < 4; i++ {
		px.Pix[i*4+0], px.Pix[i*4+1], px.Pix[i*4+2], px.Pix[i*4+3] = 255, 0, 0, 255
	}
	if err := png.Encode(&buf, px); err != nil {
		t.Fatal(err)
	}
	href := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	svg := `<svg viewBox="0 0 4 4"><defs>` +
		`<pattern id="pat" patternContentUnits="objectBoundingBox" width="1" height="1">` +
		`<use href="#img"/>` +
		`</pattern>` +
		`<image id="img" width="4" height="4" href="` + href + `"/>` +
		`</defs><rect width="4" height="4" fill="url(#pat)"/></svg>`
	img, err := Render([]byte(svg), 16)
	if err != nil {
		t.Fatal(err)
	}
	if n := nonTransparent(img); n == 0 {
		t.Fatal("Figma-style pattern (pattern+use+data-uri image) should render")
	}
}

func TestPatternElementIsNotRenderedDirectly(t *testing.T) {
	// pattern 直接挂在 svg 下（不在 defs 里）也不应被当作可见元素，
	// 更不应报 unsupported element 诊断。
	svg := `<svg viewBox="0 0 4 4"><pattern id="p" width="1" height="1"><rect width="1" height="1" fill="red"/></pattern></svg>`
	doc, err := Parse([]byte(svg))
	if err != nil {
		t.Fatal(err)
	}
	img, err := doc.Render(8, 8)
	if err != nil {
		t.Fatal(err)
	}
	if n := nonTransparent(img); n != 0 {
		t.Fatalf("bare <pattern> must not render, got %d pixels", n)
	}
	for _, d := range doc.Diagnostics() {
		if strings.Contains(d, "unsupported element skipped: pattern") {
			t.Fatalf("<pattern> should not be reported unsupported: %v", doc.Diagnostics())
		}
	}
}

func TestPatternMissingReferenceDoesNotFill(t *testing.T) {
	svg := `<svg viewBox="0 0 4 4"><rect width="4" height="4" fill="url(#nope)"/></svg>`
	img, err := Render([]byte(svg), 8)
	if err != nil {
		t.Fatal(err)
	}
	if n := nonTransparent(img); n != 0 {
		t.Fatalf("missing pattern/gradient ref should not fill, got %d", n)
	}
}

func TestPatternZeroTileSizeIsDiagnosed(t *testing.T) {
	svg := `<svg viewBox="0 0 4 4"><defs><pattern id="p" width="0" height="0"><rect width="1" height="1"/></pattern></defs><rect width="4" height="4" fill="url(#p)"/></svg>`
	doc, err := Parse([]byte(svg))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.Render(8, 8); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(doc.Diagnostics(), "\n")
	if !strings.Contains(joined, "non-positive tile size") {
		t.Fatalf("missing zero-size diagnostic; got:\n%s", joined)
	}
}

func TestStrokePatternIsDiagnosed(t *testing.T) {
	svg := `<svg viewBox="0 0 4 4"><defs><pattern id="p" width="1" height="1"><rect width="1" height="1" fill="red"/></pattern></defs><rect x="0.5" y="0.5" width="3" height="3" fill="none" stroke="url(#p)" stroke-width="1"/></svg>`
	doc, err := Parse([]byte(svg))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.Render(8, 8); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(doc.Diagnostics(), "\n")
	if !strings.Contains(joined, "stroke with pattern reference") {
		t.Fatalf("missing stroke-pattern diagnostic; got:\n%s", joined)
	}
}

func TestResolvePatternRejectsNonPattern(t *testing.T) {
	doc, err := Parse([]byte(`<svg viewBox="0 0 4 4"><defs><linearGradient id="g"><stop offset="0" stop-color="red"/></linearGradient></defs><rect width="4" height="4"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	if p := doc.resolvePattern("url(#g)"); p != nil {
		t.Fatal("gradient id must not resolve as pattern")
	}
	if p := doc.resolvePattern("red"); p != nil {
		t.Fatal("plain color must not resolve as pattern")
	}
}
