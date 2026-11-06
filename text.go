package gosvg

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/Blue-Mink/gosvg/internal/rasterx"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var (
	fontOnce   sync.Once
	fontParsed *opentype.Font
)

// loadFont 惰性加载内置的 Go Regular 字体（无外部字体依赖）。
func loadFont() *opentype.Font {
	fontOnce.Do(func() {
		f, err := opentype.Parse(goregular.TTF)
		if err == nil {
			fontParsed = f
		}
	})
	return fontParsed
}

// matrixScale 近似返回当前变换的等比缩放因子（取 x/y 缩放的平均）。
func (ctx *renderContext) matrixScale() float64 {
	// Matrix2D 为 [a b c d e f]，等比缩放下 A≈D、B≈C≈0。
	a, b := ctx.matrix.A, ctx.matrix.B
	c, d := ctx.matrix.C, ctx.matrix.D
	sx := math.Hypot(a, b)
	sy := math.Hypot(c, d)
	return (sx + sy) / 2
}

// renderText 渲染 <text> 元素（仅简单单行文本，无 tspans）。
func (ctx *renderContext) renderText(n *node, opacity float64) {
	text := n.content
	if text == "" {
		return
	}
	f := loadFont()
	if f == nil {
		return
	}

	fontSize := parseFloatOr(n.attr("font-size"), 12)
	if fontSize <= 0 {
		return
	}
	anchor := n.attr("text-anchor")
	baseline := n.attr("dominant-baseline")

	c := color.NRGBA{0, 0, 0, 255}
	if v, ok := parseColor(n.attr("fill")); ok {
		c = v
	}
	fillOpacity := parseFloatOr(n.attr("fill-opacity"), 1)

	x := parseFloatOr(n.attr("x"), 0)
	y := parseFloatOr(n.attr("y"), 0)

	// 变换到像素坐标
	px, py := ctx.matrix.Transform(x, y)
	scale := ctx.matrixScale()
	if scale <= 0 {
		scale = 1
	}

	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    fontSize * scale,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return
	}
	defer face.Close()

	// 测量宽度（fixed.Int26_6 → px）
	widthPx := float64(font.MeasureString(face, text)) / 64

	switch anchor {
	case "middle":
		px -= widthPx / 2
	case "end":
		px -= widthPx
	}

	// dominant-baseline：把 y 从"基线"调整为"文字垂直中心"。
	switch baseline {
	case "central", "middle":
		m := face.Metrics()
		shift := (m.Ascent + m.Descent) / 2 // descent 为负
		py -= float64(shift) / 64
	case "text-before-edge", "hanging":
		m := face.Metrics()
		py -= float64(m.Ascent) / 64
	}

	dot := fixed.Point26_6{
		X: fixed.Int26_6(px * 64),
		Y: fixed.Int26_6(py * 64),
	}
	drawer := &font.Drawer{
		Dst:  ctx.img,
		Src:  image.NewUniform(rasterx.ApplyOpacity(c, opacity*fillOpacity*ctx.opacity)),
		Face: face,
		Dot:  dot,
	}
	drawer.DrawString(text)
}
