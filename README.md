# gosvg

纯 Go 的轻量级 SVG 子集渲染器。零 cgo，唯一直接依赖 `golang.org/x/image`（Go 官方扩展库），
把最常见的 SVG 场景渲染成 `*image.NRGBA` 位图。

## 定位

不是完整 SVG 1.1 实现，而是"够用的 SVG 子集"。目标场景：

- **circle-flags 圆形国旗**（`mask` + `fill` + `path`）
- **Lucide / Feather 线条图标**（`path` + `stroke`）
- **带简单文字的图标**（`text`）

## 特性

**元素**：`svg` / `g` / `defs` / `use` / `circle` / `ellipse` / `rect` / `line` /
`path` / `polygon` / `polyline` / `text` / `mask`

**属性**：`viewBox` / `transform` / `fill` / `fill-rule` / `fill-opacity` /
`stroke` / `stroke-width` / `stroke-linecap` / `stroke-linejoin` /
`stroke-dasharray` / `stroke-dashoffset` / `stroke-opacity` /
`opacity` / `font-size` / `font-family` / `font-weight` / `font-style` /
`text-anchor` / `dominant-baseline` / `color` / `currentColor` / `id` / `mask` /
内联 `style`（CSS 声明，优先级高于 presentation 属性）

**明确不支持**（命中时静默降级，不报错）：gradient、clip-path、tspan、`<style>` 样式表、
class 选择器、`preserveAspectRatio` 非默认值、`mask-type="alpha"`。

## 安装

```sh
go get github.com/Blue-Mink/gosvg
```

## 用法

```go
import "github.com/Blue-Mink/gosvg"

svg := []byte(`<svg viewBox="0 0 24 24">
  <path fill="none" stroke="#000" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"/>
</svg>`)

// 渲染成 512×512 的透明底 NRGBA
img, err := gosvg.Render(svg, 512)
```

主要 API：

| 函数 | 说明 |
|---|---|
| `Render(data []byte, size int)` | 渲染成 `size×size` 位图 |
| `RenderTo(data []byte, w, h int)` | 渲染成指定宽高 |
| `Parse(data []byte)` | 解析为可复用的 `*Document` |
| `(*Document).Render(w, h int)` | 对已解析文档多次渲染 |
| `AddFontDir(dir string) error` | 注册系统字体目录（可选，见下） |

## 字体

默认用内置 Go Regular 字体（零外部依赖）。如需对齐系统字体（如 DejaVu），
可选注册字体目录：

```go
gosvg.AddFontDir("/usr/share/fonts/truetype/dejavu")
```

之后 `font-family`（含 CSS 通用名 `sans-serif`/`serif`/`monospace` 与逗号列表）
会命中对应字型，`font-weight`/`font-style` 择优；未命中回退内置字体。
不调用 `AddFontDir` 则保持零依赖行为不变。

## 与 cairosvg 对账（512×512，IoU）

| 场景 | IoU |
|---|---|
| 圆形国旗（26 国，fill + mask） | 0.9998 |
| Lucide 线条图标（rock/country/blues/jazz） | 0.9946–0.9985 |
| 年代唱片标签（text） | 字形一致（DejaVu 对齐） |

剩余差异为光栅器 AA 与 cairosvg 的亚像素舍入差，肉眼不可辨。

## 许可证

本项目自研代码遵循 [MIT License](LICENSE)。

第三方 fork 代码保留各自原始版权与许可：

- `internal/rasterx` —— fork 自 [srwiley/rasterx](https://github.com/srwiley/rasterx)，
  MIT，© 2018 the rasterx Authors，见 `internal/rasterx/LICENSE`
- `internal/vector` —— fork 自 [`golang.org/x/image/vector`](https://pkg.go.dev/golang.org/x/image/vector)，
  BSD-3-Clause，© The Go Authors，见 `internal/vector/LICENSE`
