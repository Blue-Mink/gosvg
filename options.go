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
	// ImageDecoder 是本次渲染专用的 <image> 解码器；nil 时回退到包级 ImageDecoder。
	// 放进选项可避免全局状态（见 DefaultImageDecoder）。
	ImageDecoder func(string) (image.Image, error)
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
		return &UnsupportedError{Feature: "non-positive size", Code: CodeNonPositiveSize, Err: ErrNonPositiveSize}
	}
	if m := o.maxDim(); w > m || h > m {
		return &UnsupportedError{
			Feature: fmt.Sprintf("size %dx%d exceeds MaxDimension %d", w, h, m),
			Code:    CodeDimensionExceeded,
			Err:     ErrDimensionExceeded,
		}
	}
	return nil
}

// fillBackground 用背景色填满整张图（只在渲染前调用一次，不改变尺寸）。
func fillBackground(img *image.NRGBA, c color.Color) {
	// 用 NRGBAModel 转换，避免把预乘的 color.RGBA 当成非预乘 NRGBA 直接写入
	// 导致半透明背景色偏。
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	px := img.Pix
	for i := 0; i < len(px); i += 4 {
		px[i], px[i+1], px[i+2], px[i+3] = n.R, n.G, n.B, n.A
	}
}
