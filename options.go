package gosvg

import (
	"fmt"
	"image"
	"image/color"
)

const (
	// DefaultMaxDimension 是单次渲染宽/高的默认上限（像素），防止超大尺寸触发
	// 巨量位图分配导致 OOM。可通过 RenderOptions.MaxDimension 调整。
	DefaultMaxDimension = 8192

	// maxNestingDepth 限制 mask / clipPath 等离屏渲染的嵌套层数，避免多层叠加
	// 各分配一份全尺寸 NRGBA 造成的乘法级内存占用。
	maxNestingDepth = 32
)

// RenderOptions 控制渲染行为。零值等价于 DefaultRenderOptions()。
type RenderOptions struct {
	// MaxDimension 是宽/高的上限；<= 0 表示使用 DefaultMaxDimension。
	MaxDimension int
	// Background 非 nil 时先填充该颜色作为底色；nil 保持透明底（默认）。
	Background color.Color
}

// DefaultRenderOptions 返回默认选项：透明底、DefaultMaxDimension 尺寸上限。
func DefaultRenderOptions() RenderOptions {
	return RenderOptions{MaxDimension: DefaultMaxDimension}
}

func (o RenderOptions) maxDim() int {
	if o.MaxDimension > 0 {
		return o.MaxDimension
	}
	return DefaultMaxDimension
}

func validateSize(w, h int, o RenderOptions) error {
	if w <= 0 || h <= 0 {
		return &UnsupportedError{"non-positive size"}
	}
	if m := o.maxDim(); w > m || h > m {
		return &UnsupportedError{fmt.Sprintf("size %dx%d exceeds MaxDimension %d", w, h, m)}
	}
	return nil
}

// fillBackground 用背景色填满整张图（只在渲染前调用一次，不改变尺寸）。
func fillBackground(img *image.NRGBA, c color.Color) {
	r, g, b, a := c.RGBA()
	rr, gg, bb, aa := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)
	px := img.Pix
	for i := 0; i < len(px); i += 4 {
		px[i], px[i+1], px[i+2], px[i+3] = rr, gg, bb, aa
	}
}
