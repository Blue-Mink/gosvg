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
//	     clipPath / circle / ellipse / rect / line / path / polygon /
//	     polyline / text / tspan / mask / marker / image / style
//	属性：viewBox / preserveAspectRatio / transform / fill / fill-rule / fill-opacity /
//	     stroke / stroke-width / stroke-linecap / stroke-linejoin /
//	     stroke-miterlimit / stroke-dasharray / stroke-dashoffset /
//	     stroke-opacity / vector-effect（non-scaling-stroke）/ opacity / gradientUnits / gradientTransform /
//	     spreadMethod / offset / stop-color / stop-opacity /
//	     font-size / font-family / font-weight / font-style /
//	     text-anchor / dominant-baseline / color / currentColor /
//	     id / class / mask / maskUnits / maskContentUnits /
//	     clip-path / clipPathUnits /
//	     marker-start / marker-mid / marker-end /
//	     markerUnits / markerWidth / markerHeight / refX / refY / orient /
//	     x / y / dx / dy（text 与 tspan 定位）/
//	     style（内联 CSS 声明，优先级高于 presentation 属性）
//
// 文字字体：默认用内置 Go Regular（零外部依赖）。可选调用 AddFontDir 注册
// 系统字体目录（如 /usr/share/fonts/truetype/dejavu），命中 font-family
// （含 CSS 通用名 sans-serif/serif/monospace 与逗号列表）则换用对应字型，
// 字重/字型按 font-weight/font-style 择优；未命中回退内置字体。
//
// 颜色格式：支持 #rgb / #rrggbb / #rgba / #rrggbbaa、rgb()/rgba()、
// hsl()/hsla()（逗号或空格分隔，含 "/" 分隔 alpha，h 支持 deg 单位）、
// SVG/CSS 命名颜色、none、currentColor。
//
// 明确不支持的常见特性（命中时静默跳过，不报错）：
//
//	元素/特性：filter、textPath、foreignObject、symbol/view、
//	    switch/requiredFeatures、动画与脚本、mask-type="alpha"、
//	    （vector-effect="non-scaling-stroke"、marker orient="auto-start-reverse"、
//	    <image> 自身的 preserveAspectRatio、radialGradient 的 fx/fy 均已支持）、
//	    mix-blend-mode/isolation、letter-spacing/word-spacing/text-decoration、
//	    use 的 width/height（fr 属 SVG 2，未实现）
//	    （clip-rule 已支持：clipPath 内映射到 fill-rule 的 even-odd 光栅路径）
//	CSS：后代/子代/兄弟选择器、伪类、@media/@import/@supports、!important、
//	    var()/calc()、inherit 关键字、CSS4 颜色（lab/lch/color/device-cmyk）
//	长度：根 <svg> 的 width/height 按 96dpi 换算物理单位；其余长度统一剥离后
//	    按数值 1:1 近似，不做物理换算，% 不按基准解析
//	字体：.ttc 集合；未调用 AddFontDir 时 font-family 一律回退 Go Regular
//	<image>：data: URI 内置解码；DefaultImageDecoder()（仅 data: URI）需显式注入，
//	    本地路径需显式用 LocalFileImageDecoder(roots...)；也可用包级 ImageDecoder /
//	    RenderOptions.ImageDecoder 覆盖；两者皆空时静默跳过
//
// 完整清单见 GOSVG_TODO.md / README.md。渲染期跳过事件可通过
// Document.Diagnostics() / LastRenderDiagnostics() 观察（只读通道，不改变渲染结果）；
// 该通道覆盖主要跳过分支，未覆盖项与各轮处置结论见 GOSVG_GAP_TODO.md 档 2 / 档 3。
package gosvg

import (
	"context"
	"errors"
	"image"
	"sync"
)

// Render 把 SVG 数据渲染成一张 size×size 的透明底 NRGBA 位图。
// 内部按 viewBox 等比缩放；SVG 没有 viewBox 时按 width/height 回退。
func Render(data []byte, size int) (*image.NRGBA, error) {
	return RenderTo(data, size, size)
}

// RenderTo 把 SVG 数据渲染成 width×height 的透明底 NRGBA 位图。
func RenderTo(data []byte, width, height int) (*image.NRGBA, error) {
	return RenderWithOptions(context.Background(), data, width, height, DefaultRenderOptions())
}

// RenderWithOptions 用 ctx 与 opts 渲染 SVG。ctx 取消时返回 ctx.Err()；
// opts 为零值时等价于默认选项。旧的 Render / RenderTo 内部转调本函数，行为不变。
func RenderWithOptions(ctx context.Context, data []byte, width, height int, opts RenderOptions) (*image.NRGBA, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateSize(width, height, opts); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return doc.render(ctx, width, height, opts)
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
	root     *node
	viewBox  [4]float64
	hasVB    bool
	width    float64
	height   float64
	defs     map[string]*node
	grads    map[string]*node
	par      preserveAspect // preserveAspectRatio 解析结果
	cssRules []cssRule      // <style> 元素解析出的 CSS 规则

	diagMu    sync.Mutex      // 保护下面所有诊断字段
	diagSeen  map[string]bool // 累计去重
	diags     []string        // 渲染期跳过事件（累计）
	lastSeen  map[string]bool // 最近一次渲染去重
	lastDiags []string        // 最近一次渲染的跳过事件
}

// Render 把文档渲染成 width×height 的 NRGBA 位图（默认选项）。
func (d *Document) Render(width, height int) (*image.NRGBA, error) {
	return d.RenderWithOptions(context.Background(), width, height, DefaultRenderOptions())
}

// RenderWithOptions 用 ctx 与 opts 渲染已解析的文档，可对不同尺寸多次调用。
func (d *Document) RenderWithOptions(ctx context.Context, width, height int, opts RenderOptions) (*image.NRGBA, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateSize(width, height, opts); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return d.render(ctx, width, height, opts)
}

func (d *Document) render(ctx context.Context, width, height int, opts RenderOptions) (*image.NRGBA, error) {
	d.resetLastDiagnostics()
	rctx := newRenderContext(width, height, d)
	rctx.ctx = ctx
	rctx.opts = opts
	if opts.Background != nil {
		fillBackground(rctx.img, opts.Background)
	}
	if err := rctx.renderNode(d.root, 1.0); err != nil {
		return nil, err
	}
	if rctx.err != nil {
		return nil, rctx.err
	}
	clearGhostRGB(rctx.img)
	return rctx.img, nil
}

// addDiagnostic 记录一次渲染期跳过事件（按文本去重，并发安全）。
func (d *Document) addDiagnostic(s string) {
	d.diagMu.Lock()
	defer d.diagMu.Unlock()
	if d.diagSeen == nil {
		d.diagSeen = map[string]bool{}
	}
	if !d.diagSeen[s] {
		d.diagSeen[s] = true
		d.diags = append(d.diags, s)
	}
	if d.lastSeen == nil {
		d.lastSeen = map[string]bool{}
	}
	if !d.lastSeen[s] {
		d.lastSeen[s] = true
		d.lastDiags = append(d.lastDiags, s)
	}
}

// resetLastDiagnostics 在每次 Render 开头清空「最近一次渲染」的诊断。
func (d *Document) resetLastDiagnostics() {
	d.diagMu.Lock()
	defer d.diagMu.Unlock()
	d.lastSeen = nil
	d.lastDiags = nil
}

// Diagnostics 返回本 Document 渲染过程中遇到的跳过事件（按文本去重）。
// 这是只读观察通道，不改变渲染结果；事件为累计值，多次 Render 不清空。
// 覆盖主要跳过点（未识别元素、image/mask/clip/use/marker/gradient 引用缺失、
// 嵌套超限），并非穷尽所有静默分支。若只需最近一次渲染的事件，见
// LastRenderDiagnostics()。
func (d *Document) Diagnostics() []string {
	d.diagMu.Lock()
	defer d.diagMu.Unlock()
	out := make([]string, len(d.diags))
	copy(out, d.diags)
	return out
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

// ErrorCode 是 UnsupportedError 的机器可读分类。
type ErrorCode string

const (
	CodeEmptyDocument     ErrorCode = "empty_document"
	CodeNonPositiveSize   ErrorCode = "non_positive_size"
	CodeDimensionExceeded ErrorCode = "dimension_exceeded"
	CodeInputTooLarge     ErrorCode = "input_too_large"
	CodeTooManyElements   ErrorCode = "too_many_elements"
	CodeNestingTooDeep    ErrorCode = "nesting_too_deep"
)

// 哨兵错误，供 errors.Is / errors.As 判断。
var (
	ErrEmptyDocument     = errors.New("gosvg: empty document")
	ErrNonPositiveSize   = errors.New("gosvg: non-positive size")
	ErrDimensionExceeded = errors.New("gosvg: dimension exceeded")
	ErrInputTooLarge     = errors.New("gosvg: SVG input too large")
	ErrTooManyElements   = errors.New("gosvg: too many SVG elements")
	ErrNestingTooDeep    = errors.New("gosvg: XML nesting too deep")
)

// UnsupportedError 表示调用方传入了本库无法处理的输入（空文档、非正尺寸、尺寸超限）。
// Code 供程序判断，Err 供 errors.Is，Feature 保留人类可读文本（向后兼容）。
// 注意：超出子集的 SVG 特性不会返回本错误，而是静默跳过（见包注释「明确不支持」）。
type UnsupportedError struct {
	Feature string
	Code    ErrorCode
	Err     error
}

func (e *UnsupportedError) Error() string { return "gosvg: unsupported " + e.Feature }
func (e *UnsupportedError) Unwrap() error { return e.Err }

// LastRenderDiagnostics 返回最近一次 Render 的跳过事件（每次渲染开头重置）。
// 与累计的 Diagnostics() 互补：复用 Document 多次渲染时用它区分「这一次」。
func (d *Document) LastRenderDiagnostics() []string {
	d.diagMu.Lock()
	defer d.diagMu.Unlock()
	out := make([]string, len(d.lastDiags))
	copy(out, d.lastDiags)
	return out
}
