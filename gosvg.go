// Package gosvg 是一个纯 Go 的轻量级 SVG 子集渲染器。
//
// 它定位为"够用的 SVG 子集"，而非完整 SVG 1.1 实现：目标是以零 cgo、
// 零外部运行依赖，正确渲染实际场景中最常见的图形——circle-flags 风格的
// 圆形国旗（mask）、Lucide/Feather 风格的线条图标（path + stroke）、以及
// 带简单文字的图标（text）。
//
// 支持的 SVG 子集：
//
//	元素：svg / g / defs / use / linearGradient / radialGradient / stop /
//	     circle / ellipse / rect / line / path / polygon / polyline /
//	     text / mask
//	属性：viewBox / transform / fill / fill-rule / fill-opacity /
//	     stroke / stroke-width / stroke-linecap / stroke-linejoin /
//	     stroke-miterlimit / stroke-dasharray / stroke-dashoffset /
//	     stroke-opacity / opacity / gradientUnits / gradientTransform /
//	     spreadMethod / offset / stop-color / stop-opacity /
//	     font-size / font-family / font-weight / font-style /
//	     text-anchor / dominant-baseline / color / currentColor /
//	     id / mask / style（内联 CSS 声明，优先级高于 presentation 属性）
//
// 文字字体：默认用内置 Go Regular（零外部依赖）。可选调用 AddFontDir 注册
// 系统字体目录（如 /usr/share/fonts/truetype/dejavu），命中 font-family
// （含 CSS 通用名 sans-serif/serif/monospace 与逗号列表）则换用对应字型，
// 字重/字型按 font-weight/font-style 择优；未命中回退内置字体。
//
// 明确不支持的常见特性（命中时静默降级，不报错）：
//
//	clip-path / tspan
//	class 选择器 / <style> 样式表元素
//	preserveAspectRatio 的非默认值（仅支持 xMidYMid meet）
//	mask-type="alpha"（仅支持默认 luminance）
//	fill-rule="evenodd"（底层 vector 光栅器仅支持 non-zero winding，evenodd 降级为 non-zero）
package gosvg

import (
	"image"
)

// Render 把 SVG 数据渲染成一张 size×size 的透明底 NRGBA 位图。
// 内部按 viewBox 等比缩放；SVG 没有 viewBox 时按 width/height 回退。
func Render(data []byte, size int) (*image.NRGBA, error) {
	return RenderTo(data, size, size)
}

// RenderTo 把 SVG 数据渲染成 width×height 的透明底 NRGBA 位图。
func RenderTo(data []byte, width, height int) (*image.NRGBA, error) {
	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return doc.Render(width, height)
}

// Parse 解析 SVG 数据，返回可复用的 Document（可对不同尺寸多次渲染）。
func Parse(data []byte) (*Document, error) {
	root, err := parse(data)
	if err != nil {
		return nil, err
	}
	doc := &Document{root: root}
	if err := doc.resolve(); err != nil {
		return nil, err
	}
	return doc, nil
}

// Document 是一份已解析的 SVG 文档。
type Document struct {
	root    *node
	viewBox [4]float64
	hasVB   bool
	width   float64
	height  float64
	defs    map[string]*node
	grads   map[string]*node
	// err 记录解析期遇到的、可安全忽略的问题（供调试），不影响渲染。
}

// Render 把文档渲染成 width×height 的 NRGBA 位图。
func (d *Document) Render(width, height int) (*image.NRGBA, error) {
	if width <= 0 || height <= 0 {
		return nil, &UnsupportedError{"non-positive size"}
	}
	ctx := newRenderContext(width, height, d)
	if err := ctx.renderNode(d.root, 1.0); err != nil {
		return nil, err
	}
	clearGhostRGB(ctx.img)
	return ctx.img, nil
}

// clearGhostRGB 清除 alpha=0 像素上的残留 RGB。
// 通用光栅化路径对微量覆盖像素做「RGB÷极小alpha」放大后，uint8 存储会截断
// alpha 为 0 却保留被放大的 RGB；虽然视觉不可见，但会污染下游读 RGB 的逻辑
// 与非透明底合成。渲染收尾统一清零。
func clearGhostRGB(img *image.NRGBA) {
	px := img.Pix
	for i := 3; i < len(px); i += 4 {
		if px[i] == 0 {
			px[i-3], px[i-2], px[i-1] = 0, 0, 0
		}
	}
}

// UnsupportedError 表示遇到了超出本库子集的 SVG 特性（非致命，已跳过）。
type UnsupportedError struct{ Feature string }

func (e *UnsupportedError) Error() string { return "gosvg: unsupported " + e.Feature }
