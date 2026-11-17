package gosvg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// solidDataURI 生成 w×h 纯色 PNG 的 data URI。
func solidDataURI(w, h int, c color.NRGBA) string {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// countColor 统计画布内指定颜色像素数。
func countColor(img *image.NRGBA, c color.NRGBA) int {
	n := 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			c2 := img.NRGBAAt(x, y)
			if c2.A == c.A && c2.R == c.R && c2.G == c.G && c2.B == c.B {
				n++
			}
		}
	}
	return n
}

// 2:1 的红色 src 放进 1:1 的目标（40×40）。
// meet：等比缩放至宽 40、高 20，上下各留 10px 空白 → 红色像素约 40*20=800。
// slice：等比缩放至高 40、宽 80，左右各裁 20px → 红色像素约 40*40=1600（铺满）。
const imgPARmeet = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><image x="30" y="30" width="40" height="40" preserveAspectRatio="xMidYMid meet" href="%s"/></svg>`
const imgPARslice = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><image x="30" y="30" width="40" height="40" preserveAspectRatio="xMidYMid slice" href="%s"/></svg>`
const imgPARnone = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><image x="30" y="30" width="40" height="40" preserveAspectRatio="none" href="%s"/></svg>`

func renderPAR(t *testing.T, tmpl, uri string) *image.NRGBA {
	data := []byte(fmt.Sprintf(tmpl, uri))
	img, err := Render(data, 100)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return img
}

func TestImagePreserveAspectRatio(t *testing.T) {
	red := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	uri := solidDataURI(20, 10, red) // 2:1

	meetN := countColor(renderPAR(t, imgPARmeet, uri), red)
	sliceN := countColor(renderPAR(t, imgPARslice, uri), red)
	noneN := countColor(renderPAR(t, imgPARnone, uri), red)
	t.Logf("meet=%d slice=%d none=%d", meetN, sliceN, noneN)

	// meet 应留白（红色明显少于铺满）；slice 与 none 都应铺满 40×40=1600。
	if meetN >= sliceN*7/10 {
		t.Errorf("meet 应等比留白: meet=%d slice=%d", meetN, sliceN)
	}
	if sliceN < 1500 || noneN < 1500 {
		t.Errorf("slice/none 应铺满目标: slice=%d none=%d", sliceN, noneN)
	}

	// xMidYMid meet：目标 30..70，2:1 src 等比缩放为 40×20，垂直居中 →
	// 红色集中在 y=40..60 的中带，上下（30..40、60..70）应留白。
	img := renderPAR(t, imgPARmeet, uri)
	topWhite := countColorIn(img, red, 30, 70, 31, 39)
	midRed := countColorIn(img, red, 30, 70, 42, 58)
	botWhite := countColorIn(img, red, 30, 70, 61, 69)
	if topWhite != 0 || botWhite != 0 || midRed < 600 {
		t.Errorf("xMidYMid meet 应垂直居中留白: top=%d mid=%d bot=%d", topWhite, midRed, botWhite)
	}
}

func countColorIn(img *image.NRGBA, c color.NRGBA, x0, x1, y0, y1 int) int {
	n := 0
	for y := y0; y < y1 && y < img.Bounds().Dy(); y++ {
		for x := x0; x < x1 && x < img.Bounds().Dx(); x++ {
			c2 := img.NRGBAAt(x, y)
			if c2.A == c.A && c2.R == c.R && c2.G == c.G && c2.B == c.B {
				n++
			}
		}
	}
	return n
}
