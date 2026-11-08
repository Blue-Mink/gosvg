# GOSVG — 纯 Go 轻量 SVG 子集渲染库

> 本文件是**当前状态**的单一事实来源：支持范围、明确不支持、已知边界、质量基线、变更记录。
> 历史施工过程与已修复 bug 的细节见 Git 提交历史（`git log`）；本文件不再保留过时的
> 「待办/未实现」快照。尚未决策或需要外部条件的缺口见
> [`GOSVG_GAP_TODO.md`](GOSVG_GAP_TODO.md)。

## 目标与定位

可开源、可维护、定位清晰的纯 Go SVG 子集渲染器：零 cgo、运行时零外部依赖（唯一直接依赖
`golang.org/x/image`），输出 `*image.NRGBA`。覆盖 circle-flags 圆形国旗（mask）、Lucide 线条
图标（path + stroke）、年代唱片标签（text）三类实际场景。**明确不做完整 SVG 1.1**。

## 当前支持范围

**元素**：`svg` / `g` / `defs` / `use` / `linearGradient` / `radialGradient` / `stop` /
`clipPath` / `circle` / `ellipse` / `rect` / `line` / `path` / `polygon` / `polyline` /
`text` / `tspan` / `mask` / `marker` / `image` / `style`

**属性**：`viewBox` / `preserveAspectRatio`（9 种 align + meet/slice/none）/ `transform`
（matrix/translate/scale/rotate/skewX/skewY）/ `fill` / `fill-rule` / `fill-opacity` /
`stroke` / `stroke-width` / `stroke-linecap` / `stroke-linejoin` / `stroke-miterlimit` /
`stroke-dasharray` / `stroke-dashoffset` / `stroke-opacity` / `opacity` /
`gradientUnits` / `gradientTransform` / `spreadMethod` / `offset` / `stop-color` /
`stop-opacity` / `mask` / `maskUnits` / `maskContentUnits` / `clip-path` / `clipPathUnits` /
`marker-start` / `marker-mid` / `marker-end` / `markerUnits` / `markerWidth` / `markerHeight` /
`refX` / `refY` / `orient` / `font-size` / `font-family` / `font-weight` / `font-style` /
`text-anchor` / `dominant-baseline` / `x` / `y` / `dx` / `dy` / `color` / `currentColor` /
`id` / `class` / 内联 `style`

**CSS**：`<style>` 样式表 + class/id/类型选择器（含 `tag.class#id` 组合、逗号分隔），按
(id, class, tag) 特异性级联；优先级 presentation 属性 < CSS 规则 < 内联 `style`。

**颜色**：`#rgb` / `#rrggbb` / `#rgba` / `#rrggbbaa`、`rgb()` / `rgba()`、`hsl()` / `hsla()`
（逗号或空格分隔，含 `/` 分隔 alpha）、SVG/CSS 命名颜色、`none`、`currentColor`。

**文本**：内置 Go Regular（零依赖）；可选 `AddFontDir(dir)` 注册系统字体目录，命中
`font-family`（含 CSS 通用名与逗号列表）后按 `font-weight` / `font-style` 择优；未命中回退。
`MeasureText` / `MeasureTextEx` 供手动布局。

**底层**：`encoding/xml` 解析 → 元素树 → `viewBox` 坐标系 → fill/stroke 光栅化
（`internal/rasterx` 扫描器 + `internal/vector` AA 光栅器）+ luminance mask 合成 +
形状 alpha 裁剪 + opentype 文字，汇到同一张位图。
stroke 采用「每条 flatten 线段独立生成胶囊体（偏移矩形 + 两端半圆）+ non-zero 累积」的
精确轮廓方案，小半径弧内偏移退化天然免疫。`fill-rule="evenodd"` 由 fork 的 vector 光栅器
补 even-odd winding 实现。

## 明确不支持（命中时静默跳过，不报错）

**元素/特性**：`filter`、`pattern`、`textPath`、`foreignObject`、`symbol` / `view`、
`switch` / `requiredFeatures`、动画与脚本、`mask-type="alpha"`、`clip-rule`、
`vector-effect`（含 `non-scaling-stroke`）、`marker orient="auto-start-reverse"`、
`radialGradient` 的 `fx/fy/fr`、`mix-blend-mode` / `isolation`、
`letter-spacing` / `word-spacing` / `text-decoration`、`use` 的 `width/height` 语义、
`<image>` 自身的 `preserveAspectRatio`（按 x/y/w/h 拉伸）。

**CSS**：后代/子代/兄弟选择器、伪类/伪元素、`@media` / `@import` / `@supports`、
`!important`、CSS 自定义属性 `var()` / `calc()`、`inherit` 关键字、CSS4 颜色
（`lab()` / `lch()` / `color()` / `device-cmyk()`）。

**长度**：所有单位（`px/pt/pc/cm/mm/in/em/ex/%` 等）统一剥离后按**数值部分 1:1 近似**，
不做物理单位换算；`%` 不按父值/视口基准解析。未知后缀（如 `12abc`）视为非法。
详见 `style.go:parseLength`。

**字体**：`.ttc` 字体集合；未调用 `AddFontDir` 时 `font-family` 一律回退 Go Regular。

**XML 编码声明**：UTF-8 与 ASCII 系（`US-ASCII` 等）透传，`ISO-8859-1` / `Latin-1`
逐字节展开为 UTF-8，**其余编码返回带编码名的明确错误**。需要完整 IANA 编码支持时，
替换包级 `CharsetReader`（例如接 `golang.org/x/text/encoding/htmlindex`）。

**`<image>`**：库不内置位图解码，需调用方注入 `ImageDecoder func(href string) (image.Image, error)`；
未注入或解码失败时静默跳过。

> 渲染期跳过事件目前**没有运行时诊断通道**（`UnsupportedError` 仅用于空文档/非正尺寸等
> 调用方输入错误）。补齐计划见 `GOSVG_GAP_TODO.md`。

## 渲染选项与诊断

- `RenderWithOptions(ctx, data, w, h, opts)` / `Document.RenderWithOptions(...)`：
  支持 `context` 取消；`RenderOptions{MaxDimension, Background}`。**旧的
  `Render` / `RenderTo` / `Document.Render` 内部转调新入口，默认行为不变**。
- `DefaultMaxDimension = 8192`：宽/高超限返回 `UnsupportedError`；
  `maxNestingDepth = 32` 限制 mask / clipPath 离屏嵌套，防止乘法级位图分配。
- `Document.Diagnostics() []string`：累计去重的跳过事件；新增
  `Document.LastRenderDiagnostics()` 返回最近一次渲染的事件（每次渲染重置）。
  覆盖未识别元素、image / mask / clip / use / marker / gradient 引用缺失、嵌套超限；
  合法的元数据元素（`title` / `desc` / `metadata` / `namedview`）不记诊断。
  这是**只读观察通道**，不改变渲染结果；未覆盖全部静默分支，实现见
  `gosvg.go` / `render.go` / `image.go`。
- `RenderWithOptions` 的 `ctx` 传 nil 时自动回退 `context.Background()`，不会 panic；
  mask / clip 离屏渲染中的取消也会回传为错误。
- **输入与资源上限**：`MaxSVGSize`（8 MiB）、`MaxElementCount`（10 万）、
  `MaxXMLDepth`（256）限制解析阶段；`DefaultMaxDimension`（8192）与
  `maxNestingDepth`（32）限制渲染阶段。超限一律返回 `UnsupportedError`
  （带 `Code` 与哨兵错误，支持 `errors.Is`）。
- **CSS 规模上限**：`MaxCSSRules`（10000）、`MaxCSSSelectorLen`（512）、
  `MaxCSSDeclLen`（4096）。与上面的硬失败不同，CSS 超限是**降级**——丢弃超限内容、
  记入 `Diagnostics()`，渲染照常完成。
- **`display` / `visibility`**：`display:none` 跳过整棵子树；`visibility` 是继承属性，
  `hidden` / `collapse` 跳过本节点绘制、子元素可显式 `visible` 覆盖。
  presentation 属性、CSS 规则、内联 `style` 三条路径都生效。
- **`<image>` 解码**：`DefaultImageDecoder()` 用标准库 PNG/JPEG/GIF 解码 `data:` URI 与
  本地路径（零外部依赖）；`RenderOptions.ImageDecoder` 支持逐渲染注入，
  优先级高于包级 `ImageDecoder`。两者都为 nil 时仍静默跳过，**默认行为不变**。
- **错误分类**：`UnsupportedError` 带 `Code`（`ErrorCode`）与 `Err`，
  提供 `ErrEmptyDocument` / `ErrNonPositiveSize` / `ErrDimensionExceeded` 供 `errors.Is`。
- **字体**：`RegisteredFonts()` 返回已注册字体族（排序副本）；`ResetFonts()` 清空注册，
  供测试隔离。

## 已知边界

1. **45° 斜线 AA 相位歧义**：`internal/vector`（fork `x/image/vector`，libart 扫描线 AA）
   对恰好穿过像素角的 45° 斜线有 ±1px 覆盖歧义，fill 与 stroke 同源受影响。
   曾尝试改坐标量化为截断：菱形 0.9698→0.9988，但 cap_butt/cap_round 从 1.0 掉到
   0.9897，引入更多回归，已回退。真实图标（Lucide 多曲线多角度）0.9946~0.9985 无显著影响；
   修复只有 4x 超采样一条路（内存/时间翻倍），收益有限 → **接受为已知边界，不修**。
2. **渲染结果取决于系统字体**：调用 `AddFontDir` 后同一 SVG 在不同机器字节不同；
   不调用则保持零依赖确定性。
3. **剩余对账差异**：rasterx AA 与 cairosvg 的亚像素舍入差（红绿平衡各约 50px），
   4x 边缘放大肉眼不可辨。

## 质量基线

```sh
sh ./scripts/verify.sh                         # 全套门禁（含覆盖率与 benchmark 冒烟）
go test ./...                                  # 单元 + 回归 + golden 哈希
go test -count=1 -cover ./...                  # 覆盖率（根包 85.8%、cmd/verify 62.3%、
                                               #   internal/vector 24.3%、internal/rasterx 3.9%）
go test -run '^$' -bench . -benchmem -benchtime 2s .   # 性能基线，见 BENCHMARK.md
go test -race ./...                           # 并发检查
go vet ./...                                  # 静态检查
gofmt -l .                                    # 空输出 = 通过
GOARCH=arm64 GOOS=linux go build ./...         # 交叉编译（真机测试需 arm64 runner）
GOOS=js GOARCH=wasm go build ./...              # WASM 目标可编译
go test -run TestGoldenRegression -update .    # 重新生成 testdata/golden.sha256
go test -run '^$' -fuzz '^FuzzParse$' -fuzztime 30s .   # fuzz 冒烟
go run ./cmd/corpus -size 64 <dir>...                  # 真实语料回归，见 CORPUS_BASELINE.md
```

内网若 `proxy.golang.org` 不通，用 `GOPROXY=https://goproxy.cn,direct`。

与 cairosvg 在 512×512 下的 IoU（本地一次性测得的基准数据，**CI 内暂无 cairosvg 对账**）：

| 场景 | IoU | 备注 |
|---|---|---|
| 圆形国旗 cn（fill + mask） | 0.9998 | |
| path1（小半径 arc + 粗 stroke） | 0.9987 | 旧 rasterx Stroker 为 0.9443 |
| rock / circle | 0.9985 / 0.9978 | |
| country / blues / jazz | 0.9968 / 0.9962 / 0.9959 | |
| dash 直线/斜线/矩形/奇数数组/dashoffset | 1.0000 | round-cap 0.9998，曲线 0.9957 |

`testdata/golden.sha256` 是 **gosvg 自身输出的像素哈希**回归基线（防止无意改变渲染结果），
不是 cairosvg 基准；由 `go test -update` 生成。当前覆盖 6 个纯图形样例——带 `<text>` 的
`s60s` 因 `AddFontDir` 全局字体状态会随测试顺序变化，暂不纳入（见 `GOSVG_GAP_TODO.md`）。

## 变更记录

### 2026-11-07（单日 29 次提交，01:58 → 15:58）
- **stroke**：自研胶囊体并集替代 rasterx Stroker，修 5 个 winding bug；v3 锯齿优化
  （填充器改 rasterx.Filler、两处截断改四舍五入、tol 语义改像素）；path1 0.9443 → 0.9987。
- **渲染能力**：even-odd winding 真实现、gradient（fill + stroke）、mask 的
  units/region/transform、clip-path、tspan（嵌套/定位/样式覆盖）、preserveAspectRatio 完整、
  CSS `<style>` + class 选择器、hsl/现代颜色、stroke-miterlimit 语义修正、
  dasharray/dashoffset、MeasureText、marker 箭头、`<image>`（回调注入）。
- **字体**：`AddFontDir` 系统字体目录（DejaVu 对齐 cairosvg），修 dominant-baseline 方向。
- **健壮性**：审计修复 6 项（text opacity 双乘、fill 不继承、形状自身 transform/opacity
  被忽略、科学计数法切分、带单位长度解析）、`use` 循环深度上限、fuzz 5 目标 +
  修 fuzz 抓到的两个死循环（path Z 后数字、marker 循环引用）。
- **工程**：README / MIT LICENSE / 第三方许可声明、Gitea + GitHub 双 CI 门禁、仓库卫生。
- **依赖调研**：`tdewolff/canvas`（247 个 go.sum 依赖）只作算法参考，明确不引入为依赖。

## 下一步

见 [`GOSVG_GAP_TODO.md`](GOSVG_GAP_TODO.md)：按「已补齐 / 需决策 / 需外部条件」分类的
缺口清单与进度。
