package gosvg

import (
	"testing"
)

// marker 指向 +x 的箭头：path M0 0 L10 5 L0 10 z，尖端在 (10,5)=refX/refY。
// 用于 auto-start-reverse 对比：起点处箭头朝右(auto) vs 朝左(reverse)。
const asrMarkerAuto = `<marker id="m" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0 0 L10 5 L0 10 z" fill="#ff0000"/></marker>`
const asrMarkerRev = `<marker id="m" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse"><path d="M0 0 L10 5 L0 10 z" fill="#ff0000"/></marker>`

// redCentroidX 返回起点邻域内红色像素的平均 x（按 alpha 加权计数）。
func redCentroidX(t *testing.T, marker string) (float64, int) {
	data := []byte(`<svg viewBox="0 0 100 100"><defs>` + marker + `</defs>` +
		`<line x1="10" y1="50" x2="90" y2="50" stroke="black" stroke-width="1" marker-start="url(#m)"/></svg>`)
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	sum, n := 0.0, 0
	for y := 30; y < 70; y++ {
		for x := 0; x < 40; x++ {
			c := img.NRGBAAt(x, y)
			if c.A > 0 && c.R > 200 && c.G < 100 && c.B < 100 {
				sum += float64(x)
				n++
			}
		}
	}
	if n == 0 {
		return 0, 0
	}
	return sum / float64(n), n
}

func TestMarkerAutoStartReverse(t *testing.T) {
	cxAuto, nAuto := redCentroidX(t, asrMarkerAuto)
	cxRev, nRev := redCentroidX(t, asrMarkerRev)
	t.Logf("auto centroid=%.2f (n=%d), reverse centroid=%.2f (n=%d)", cxAuto, nAuto, cxRev, nRev)
	if nAuto == 0 || nRev == 0 {
		t.Fatalf("两种 orient 都应画出 marker: auto=%d reverse=%d", nAuto, nRev)
	}
	// 起点反转后箭头整体镜像到另一侧，红色重心应明显偏移（>4px）。
	if diff := cxAuto - cxRev; diff < 4 && diff > -4 {
		t.Errorf("auto-start-reverse 未改变起点箭头方向: auto=%.2f reverse=%.2f diff=%.2f", cxAuto, cxRev, diff)
	}

	// end 处 auto-start-reverse 应与 auto 完全一致（reverse 只影响 start）。
	endRender := func(marker string) int {
		data := []byte(`<svg viewBox="0 0 100 100"><defs>` + marker + `</defs>` +
			`<line x1="10" y1="50" x2="90" y2="50" stroke="black" stroke-width="1" marker-end="url(#m)"/></svg>`)
		img, err := Render(data, 100)
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		return redInRegion(img, 70, 100, 30, 70)
	}
	if a, r := endRender(asrMarkerAuto), endRender(asrMarkerRev); a != r {
		t.Errorf("auto-start-reverse 不应影响 marker-end: auto=%d reverse=%d", a, r)
	}
}
