package gosvg

import (
	"image"
	"testing"
)

const arrowMarker = `<marker id="arrow" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0 0 L10 5 L0 10 z" fill="#ff0000"/></marker>`

// redInRegion 统计 [xMin,xMax)×[yMin,yMax) 内的红色像素数。
func redInRegion(img *image.NRGBA, xMin, xMax, yMin, yMax int) int {
	n := 0
	for y := yMin; y < yMax; y++ {
		for x := xMin; x < xMax; x++ {
			c := img.NRGBAAt(x, y)
			if c.A > 0 && c.R > 200 && c.G < 100 && c.B < 100 {
				n++
			}
		}
	}
	return n
}

// TestMarkerEnd 验证 marker-end 在路径终点放置箭头。
func TestMarkerEnd(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><defs>` + arrowMarker + `</defs><line x1="10" y1="50" x2="90" y2="50" stroke="black" stroke-width="2" marker-end="url(#arrow)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if n := redInRegion(img, 70, 100, 0, 100); n == 0 {
		t.Error("marker-end 箭头应出现在终点附近")
	}
	if n := redInRegion(img, 30, 70, 0, 100); n != 0 {
		t.Error("line 中间不应有 marker")
	}
}

// TestMarkerStart 验证 marker-start 在路径起点放置箭头。
func TestMarkerStart(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><defs>` + arrowMarker + `</defs><line x1="10" y1="50" x2="90" y2="50" stroke="black" stroke-width="2" marker-start="url(#arrow)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if n := redInRegion(img, 0, 30, 0, 100); n == 0 {
		t.Error("marker-start 箭头应出现在起点附近")
	}
}

// TestMarkerMid 验证 marker-mid 在中间顶点放置箭头。
func TestMarkerMid(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><defs>` + arrowMarker + `</defs><polyline points="10,50 50,50 90,50" fill="none" stroke="black" stroke-width="2" marker-mid="url(#arrow)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// 中间顶点 (50,50) 应有 marker（x 40~60）
	if n := redInRegion(img, 40, 60, 0, 100); n == 0 {
		t.Error("marker-mid 箭头应出现在中间顶点")
	}
}

// TestMarkerNoStroke 验证无 stroke 但有 marker 时仍渲染（strokeWidth 回退为 1）。
func TestMarkerNoStroke(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 100 100"><defs>` + arrowMarker + `</defs><line x1="10" y1="50" x2="90" y2="50" stroke="none" marker-end="url(#arrow)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if n := redInRegion(img, 80, 100, 0, 100); n == 0 {
		t.Error("无 stroke 时 marker 仍应渲染（strokeWidth 回退为 1）")
	}
}
