package vector

import (
	"image"
	"image/color"
	"testing"
)

// 两个同向矩形：non-zero 会填满中心，even-odd 应把中心镂空。
// 这是 gosvg fill-rule="evenodd" 的底层实现（本 fork 对 x/image/vector 的扩展）。
func rasterize(evenOdd bool) *image.NRGBA {
	const size = 20
	z := NewRasterizer(size, size)
	z.MoveTo(2, 2)
	z.LineTo(18, 2)
	z.LineTo(18, 18)
	z.LineTo(2, 18)
	z.ClosePath()
	z.MoveTo(7, 7)
	z.LineTo(13, 7)
	z.LineTo(13, 13)
	z.LineTo(7, 13)
	z.ClosePath()

	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	src := image.NewUniform(color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	z.UseEvenOdd = evenOdd
	z.Draw(img, img.Bounds(), src, image.Point{})
	return img
}

func TestRasterizerEvenOddHole(t *testing.T) {
	eo := rasterize(true)
	if a := eo.NRGBAAt(10, 10).A; a != 0 {
		t.Fatalf("even-odd center alpha = %d, want 0 (hole)", a)
	}
	if a := eo.NRGBAAt(3, 3).A; a == 0 {
		t.Fatal("outer ring should still be filled under even-odd")
	}
}

func TestRasterizerNonZeroFillsCenter(t *testing.T) {
	nz := rasterize(false)
	if a := nz.NRGBAAt(10, 10).A; a == 0 {
		t.Fatal("non-zero winding should fill the center")
	}
}

func TestRasterizerResetIsReusable(t *testing.T) {
	z := NewRasterizer(8, 8)
	draw := func() uint8 {
		z.MoveTo(1, 1)
		z.LineTo(7, 1)
		z.LineTo(7, 7)
		z.LineTo(1, 7)
		z.ClosePath()
		img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
		z.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{R: 255, A: 255}), image.Point{})
		return img.NRGBAAt(4, 4).A
	}
	first := draw()
	z.Reset(8, 8)
	if second := draw(); second != first {
		t.Fatalf("after Reset: alpha = %d, first = %d", second, first)
	}
}
