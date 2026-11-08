<p align="center">
  <b>gosvg</b><br/>
  纯 Go 的轻量 SVG 子集渲染器，零 cgo，把常见 SVG 场景渲染成位图<br/><br/>
  <img src="https://img.shields.io/badge/Go-1.27%2B-00add8" alt="go"/>
  <img src="https://img.shields.io/badge/License-MIT-111827" alt="license"/>
  <img src="https://img.shields.io/badge/deps-x%2Fimage-3b82f6" alt="deps"/>
</p>

<p align="center">
  <a href="#快速开始">快速开始</a> ·
  <a href="#特性">特性</a> ·
  <a href="#工作原理">工作原理</a> ·
  <a href="#字体">字体</a> ·
  <a href="#对账">对账</a> ·
  <a href="#仓库结构">仓库结构</a>
</p>

---

纯 Go 的轻量级 SVG 子集渲染器：零 cgo，唯一直接依赖 `golang.org/x/image`（Go 官方扩展库）。定位是「够用的 SVG 子集」而非完整实现——覆盖 **circle-flags 圆形国旗**、**Lucide 线条图标**、**带文字的图标**这三类最常见的实际场景，渲染成 `*image.NRGBA` 位图。

## 快速开始

```sh
go get github.com/Blue-Mink/gosvg
```

> [!NOTE]
> 当前源码托管在内网 Gitea（`https://github.com/Blue-Mink/gosvg`），
> module 路径保留 `github.com/Blue-Mink/gosvg` 以便未来开源。内网/本地使用请在
> 调用方 `go.mod` 里加 `replace github.com/Blue-Mink/gosvg => <本地路径>`，
> 或配合 `GOPRIVATE` 使用；仓库尚未推送到 GitHub 时 `go get` 不可用。

```go
import "github.com/Blue-Mink/gosvg"

svg := []byte(`<svg viewBox="0 0 24 24">
  <path fill="none" stroke="#000" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"/>
</svg>`)

img, err := gosvg.Render(svg, 512) // 512×512 透明底 NRGBA
```

主要 API：

| 函数 | 说明 |
|:--|:--|
| `Render(data []byte, size int)` | 渲染成 `size×size` 位图 |
| `RenderTo(data []byte, w, h int)` | 渲染成指定宽高 |
| `Parse(data []byte)` | 解析为可复用的 `*Document` |
| `(*Document).Render(w, h int)` | 对已解析文档多次渲染 |
| `AddFontDir(dir string) error` | 注册系统字体目录（可选，见[字体](#字体)） |
| `MeasureText(text string, fontSize float64, fontFamily string) float64` | 测量文本宽度（用户单位），用于手动布局/居中 |
| `MeasureTextEx(text, fontSize, family, weight, style)` | 带 font-weight/font-style 的文本测量 |
| `ImageDecoder func(href string) (image.Image, error)` | 注入 `<image>` 位图解码（库零依赖，未注入则 `<image>` 跳过） |
| `RenderWithOptions(ctx, data, w, h, opts)` | 带取消 / 尺寸上限 / 背景色的渲染入口 |
| `RenderOptions{MaxDimension, Background}` | 渲染选项；`DefaultRenderOptions()` 取默认值 |
| `(*Document).RenderWithOptions(ctx, w, h, opts)` | 已解析文档带选项渲染 |
| `(*Document).Diagnostics() []string` | 渲染期跳过事件（去重、累计），用于排查「少了一块」 |
| `(*Document).LastRenderDiagnostics() []string` | 最近一次渲染的跳过事件（每次渲染重置） |
| `DefaultImageDecoder()` | 零外部依赖的默认 `<image>` 解码器（标准库 PNG/JPEG/GIF + data URI） |
| `RenderOptions.ImageDecoder` | 逐渲染注入解码器，避免包级全局状态 |
| `RegisteredFonts() []string` | 已注册系统字体族（排序后的副本） |
| `ResetFonts()` | 清空 `AddFontDir` 注册，用于测试隔离 |
| `ErrorCode` / `ErrEmptyDocument` 等 | `UnsupportedError` 的机器可读分类与 `errors.Is` 哨兵 |
| `RenderToWriter(w, data, width, height, format, opts)` | 渲染并按 `png` / `jpeg` 编码写入 `io.Writer` |
| `EncodeImage(w, img, format, quality, bg)` | 编码位图；JPEG 无 alpha，自动合成到 `bg`（默认白） |

## 特性

**元素**：`svg` / `g` / `defs` / `use` / `linearGradient` / `radialGradient` / `stop` / `clipPath` / `circle` / `ellipse` / `rect` / `line` / `path` / `polygon` / `polyline` / `text` / `tspan` / `mask` / `marker` / `image` / `pattern` / `style`

**属性**：`viewBox` / `preserveAspectRatio` / `transform` / `fill` / `fill-rule` / `fill-opacity` / `stroke` / `stroke-width` / `stroke-linecap` / `stroke-linejoin` / `stroke-miterlimit` / `stroke-dasharray` / `stroke-dashoffset` / `stroke-opacity` / `opacity` / `gradientUnits` / `gradientTransform` / `spreadMethod` / `offset` / `stop-color` / `stop-opacity` / `mask` / `maskUnits` / `maskContentUnits` / `clip-path` / `clipPathUnits` / `marker-start` / `marker-mid` / `marker-end` / `markerUnits` / `markerWidth` / `markerHeight` / `refX` / `refY` / `orient` / `font-size` / `font-family` / `font-weight` / `font-style` / `text-anchor` / `dominant-baseline` / `x` / `y` / `dx` / `dy` / `color` / `currentColor` / `id` / `class` / `display` / `visibility` / `clip-rule` / 内联 `style`

**CSS**：支持 `<style>` 样式表元素与 class/id/类型选择器（含 `tag.class#id` 组合与逗号分隔），按 CSS 特异性级联；优先级 presentation 属性 < CSS 规则 < 内联 `style`。

**颜色**：`#rgb` / `#rrggbb` / `#rgba` / `#rrggbbaa`、`rgb()` / `rgba()`、`hsl()` / `hsla()`（逗号或空格分隔，含 `/` 分隔 alpha）、SVG/CSS 命名颜色、`none`、`currentColor`。

> [!NOTE]
> **明确不支持**（命中时静默跳过，不报错）的特性清单：
>
> - **元素/特性**：`filter`、`textPath`、`foreignObject`、`symbol`/`view`、
>   `switch`/`requiredFeatures`、动画与脚本、`mask-type="alpha"`、
>   `vector-effect`（含 `non-scaling-stroke`）、`marker orient="auto-start-reverse"`、
>   `radialGradient` 的 `fx/fy/fr`、`mix-blend-mode`/`isolation`、
>   `letter-spacing`/`word-spacing`/`text-decoration`、`use` 的 `width/height`、
>   `<image>` 自身的 `preserveAspectRatio`（按 x/y/w/h 拉伸）
> - **CSS**：后代/子代/兄弟选择器、伪类/伪元素、`@media`/`@import`/`@supports`、
>   `!important`、`var()`/`calc()`、`inherit`、CSS4 颜色（`lab()`/`lch()`/`color()`）
> - **长度**：所有单位统一剥离后按数值 **1:1 近似**，不做物理单位换算，`%` 不按基准解析
> - **字体**：`.ttc` 集合；未调用 `AddFontDir` 时 `font-family` 一律回退 Go Regular
> - **`<image>`**：`data:` URI 用标准库内置解码、开箱可用；文件路径与远程 URL
>   需注入 `ImageDecoder`（或 `RenderOptions.ImageDecoder`），未注入时静默跳过
>
> 完整清单与已知边界见 [GOSVG_TODO.md](GOSVG_TODO.md)；渲染期跳过事件可用
> `Document.Diagnostics()` / `LastRenderDiagnostics()` 观察（非穷尽）。

> [!NOTE]
> **输入与资源限制**：单次 `Parse` 的 SVG ≤ `MaxSVGSize`（8 MiB）、元素数 ≤
> `MaxElementCount`（10 万）、XML 嵌套深度 ≤ `MaxXMLDepth`（256）；渲染尺寸 ≤
> `DefaultMaxDimension`（8192）、mask/clip 离屏嵌套 ≤ 32。超限返回 `UnsupportedError`，
> 可用 `errors.Is` 判断 `ErrInputTooLarge` / `ErrTooManyElements` / `ErrNestingTooDeep` /
> `ErrDimensionExceeded` / `ErrNonPositiveSize` / `ErrEmptyDocument`。
> `<style>` 的 CSS 另有 `MaxCSSRules` / `MaxCSSSelectorLen` / `MaxCSSDeclLen`，
> 超限内容被丢弃（降级而非失败）并记入 `Diagnostics()`。

## 工作原理

```mermaid
flowchart LR
  A["SVG 数据"] --> B["XML 解析 → 元素树"]
  B --> C["resolve：defs / gradient / mask / clipPath 索引"]
  C --> D["viewBox 等比缩放"]
  D --> E{"元素分派"}
  E -->|path / shape| F["fill + stroke（含 gradient）精确轮廓"]
  E -->|mask| G["luminance mask 合成"]
  E -->|clip-path| J["形状 alpha 裁剪"]
  E -->|text| H["opentype 字体渲染"]
  F --> I["*image.NRGBA"]
  G --> I
  J --> I
  H --> I
```

**一句话总结**：解析成元素树 → 按 `viewBox` 建立坐标系 → 形状走 fill/stroke 光栅化、`<mask>` 做亮度合成、`<text>` 用字体渲染，三者汇到同一张位图。stroke 采用「每条线段独立生成胶囊体 + non-zero 累积」的精确轮廓方案，小半径弧的内偏移退化天然免疫。

## 字体

默认用内置 Go Regular 字体（零外部依赖）。如需对齐系统字体（如 DejaVu），可选注册字体目录：

```go
gosvg.AddFontDir("/usr/share/fonts/truetype/dejavu")
```

之后 `font-family`（含 CSS 通用名 `sans-serif`/`serif`/`monospace` 与逗号列表）会命中对应字型，`font-weight`/`font-style` 择优；未命中回退内置字体。不调用 `AddFontDir` 则保持零依赖行为不变。

## 对账

与 cairosvg 在 512×512 下的 IoU（越接近 1 越一致）：

| 场景 | IoU |
|:--|:--|
| 圆形国旗（26 国，fill + mask） | 0.9998 |
| Lucide 线条图标（rock / country / blues / jazz） | 0.9946–0.9985 |
| 年代唱片标签（text） | 字形一致（DejaVu 对齐） |

剩余差异为光栅器 AA 与 cairosvg 的亚像素舍入差，肉眼不可辨。

## 仓库结构

```
gosvg.go            主 API（Render / RenderTo / Parse / Document）
parser.go           SVG 解析 → 元素树
path.go             path d 解析 → vector path
shape.go            基本形状（circle / ellipse / rect / line / polygon / polyline）
stroke.go           stroke 精确轮廓
dash.go             stroke-dasharray / dashoffset
mask.go             luminance mask 合成
clip.go             clip-path 形状 alpha 裁剪
gradient.go         linear / radial 渐变（fill 与 stroke 均支持）
text.go             text / tspan 渲染
font.go             字体（内置 Go Regular + 可选系统字体）
style.go            颜色 / 长度 / 内联样式解析
transform.go        transform / viewBox
internal/rasterx    fork 自 rasterx（fill 扫描器）
internal/vector     fork 自 x/image/vector（AA 光栅器，已扩展 even-odd winding）
cmd/render          CLI 渲染器
cmd/verify          CLI 校验
testdata/           对账测试样例
```

## 开发

```sh
sh ./scripts/verify.sh   # 全套（fmt/vet/test/race/arm64/cover/bench/fuzz）
go test ./...      # 单元 + 回归 + golden 像素哈希基线
go test -race ./...
go vet ./...
gofmt -l .         # 无输出 = 通过
GOARCH=arm64 GOOS=linux go build ./...
go test -run TestGoldenRegression -update .   # 有意改动渲染输出后重新生成基线
```

若 `proxy.golang.org` 不通，用 `GOPROXY=https://goproxy.cn,direct`。
更多见 [CONTRIBUTING.md](CONTRIBUTING.md) 与 [GOSVG_TODO.md](GOSVG_TODO.md)。

## 性能

`BenchmarkRender`（512×512，Lucide 线条图标）在 .15 / Go 1.27.3 / Ryzen 7 5825U 上实测
约 **8.9 ms/op**（同机多次 8.9–11.2 ms）、**5.37 MB/op**、**292,168 allocs/op**。
`Parse` 仅约 3.5 µs，复用 `Document` 收益可忽略；数字与机器强相关，仅作同一台机器上的
相对基线，详见 [BENCHMARK.md](BENCHMARK.md)。

## 真实语料实测

用 `cmd/corpus` 对 .15 上 **958 个真实世界 SVG**（fnOS 面板图标、node_modules 图标库、
应用配置、混合来源）批量渲染：

| 语料 | 数量 | 渲染出内容 |
|---|---|---|
| fnOS 面板图标 | 217 | **100%** |
| node_modules 图标库 | 355 | 99.7% |
| 应用配置 | 127 | **100%** |
| 混合来源 | 259 | 99.2% |
| **合计** | **958** | **99.7%** |

剩余 3 个「全透明」经逐个核查均为语料本身的空 SVG（`empty.svg`、`annotation-noicon.svg`×2），
即**可渲染的 SVG 失败数为 0**。

Parse / Render 崩溃数为 0。基线数据、剩余 4 个全透明样例的逐个定位与复现方式见
[CORPUS_BASELINE.md](CORPUS_BASELINE.md)。

```sh
go run ./cmd/corpus -size 64 <dir>...   # 文本报告
go run ./cmd/corpus -json <dir>...      # JSON，便于做基线 diff
```

## 与 cairosvg 对账

对 117 个真实语料抽样（256×256）算 alpha IoU：**中位 0.9982**、平均 0.8266、
≥0.99 的有 85 个。低分集中在 `stroke-width:.25` 的**亚像素细线** + `<text>` 字体差异
（与「45° 斜线 AA 相位歧义」同源的已定案边界），重叠区 RGB 平均差仅 3–7，
属「画在同一处但 AA/字形不同」而非语义错误。完整方法、分布、根因分类与复现步骤见
[CAIROSVG_IOU.md](CAIROSVG_IOU.md)。

> `cmd/corpus` 的成功率只能回答「有没有画出东西」，对账才能回答「画得对不对」——
> 本轮就是靠对账定位到一个成功率完全看不到的真 bug（clip/mask 与 transform 的求值顺序）。

## 📄 许可证

本项目自研代码遵循 [MIT License](LICENSE) 许可证。

第三方 fork 代码保留各自原始版权与许可：

- `internal/rasterx` —— fork 自 [srwiley/rasterx](https://github.com/srwiley/rasterx)，MIT，© 2018 the rasterx Authors，见 `internal/rasterx/LICENSE`
- `internal/vector` —— fork 自 [`golang.org/x/image/vector`](https://pkg.go.dev/golang.org/x/image/vector)，BSD-3-Clause，© The Go Authors，见 `internal/vector/LICENSE`

---

## 🙏 致谢

- [golang.org/x/image](https://pkg.go.dev/golang.org/x/image) —— font / vector 底层实现
- [srwiley/rasterx](https://github.com/srwiley/rasterx) —— fill 扫描器 fork 来源
- [tdewolff/canvas](https://github.com/tdewolff/canvas) —— stroke-dasharray 等算法参考
- [cairosvg](https://cairosvg.org) —— 渲染对账基准

---

<div align="center">

**如果觉得好用，顺手点个 ⭐ Star 支持一下！**

Made with ❤️ by [Blue-Mink](https://github.com/Blue-Mink)

</div>
