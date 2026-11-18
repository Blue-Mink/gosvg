package gosvg

import (
	"image"
	"testing"
)

// 验证 radialGradient 的 fx/fy 焦点偏移确实改变渲染。
// 两个 radialGradient 除 fx/fy 外完全一致：一个焦点居中(fx=cx)，
// 一个焦点左偏(fx=0.15)。若 fx/fy 生效，两者渲染像素应不同。
func TestRadialGradientFocalPoint(t *testing.T) {
	makeSVG := func(fx string) []byte {
		return []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">` +
			`<defs><radialGradient id="g" cx="0.5" cy="0.5" r="0.5" fx="` + fx + `" fy="0.5">` +
			`<stop offset="0" stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff"/>` +
			`</radialGradient></defs>` +
			`<rect x="0" y="0" width="100" height="100" fill="url(#g)"/></svg>`)
	}
	center := renderToNRGBA(t, makeSVG("0.5"))
	offset := renderToNRGBA(t, makeSVG("0.15"))

	diff := 0
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			a, b := center.NRGBAAt(x, y), offset.NRGBAAt(x, y)
			if a.R != b.R || a.G != b.G || a.B != b.B || a.A != b.A {
				diff++
			}
		}
	}
	t.Logf("focal fx=0.5 vs fx=0.15 差异像素数: %d / 10000", diff)
	if diff < 100 {
		t.Errorf("radialGradient fx 焦点偏移似乎未生效（差异像素仅 %d）", diff)
	}
}

func renderToNRGBA(t *testing.T, data []byte) *image.NRGBA {
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return img
}
