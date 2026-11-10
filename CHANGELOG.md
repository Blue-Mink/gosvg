# Changelog

所有 notable 变更记录在此。提交信息沿用 Conventional Commits
（`feat:` / `fix:` / `docs:` / `chore:` / `test:`）。

## [Unreleased] — 2026-11-07

首个可开源基线，同日 29 次提交。

### Added
- 纯 Go SVG 子集渲染器：`Render` / `RenderTo` / `Parse` / `Document` 主 API
- fill/stroke（胶囊体并集精确轮廓）、even-odd winding、gradient（fill + stroke）
- mask（luminance，units/region/transform）、clip-path、tspan、marker、`<image>`（回调注入）
- CSS `<style>` + class/id/类型选择器、hsl()/现代颜色、dasharray/dashoffset
- preserveAspectRatio 9 种 align + meet/slice/none、MeasureText/MeasureTextEx
- 可选系统字体目录 `AddFontDir`
- 5 个 fuzz 目标、97 个测试函数、Gitea + GitHub 双 CI 门禁
- `cmd/render` CLI、`cmd/verify` 对账工具、`testdata/` 样例

### Fixed
- path1 小半径 arc stroke 的 IoU 0.9443 → 0.9987（替换 rasterx Stroker）
- 45° 斜线 AA 相位歧义定案为已知边界（附回退实验依据）
- 审计修复 6 项渲染 bug、`use` 循环栈溢出、fuzz 抓到的两个死循环
- CSS `<style>` 的 `@media` 嵌套规则不再被误当顶层应用（沙漏整块消失，IoU 0.4549 → 0.9990）
- `clip-path` 分支不再把 `skipNodeTransform` 泄漏给子树（45° 斜线变横条，IoU 0.0677 → 0.9973）
- `<image>` 宽度超出 viewBox 时按比例取源切片，不再把整图压缩进可见区（IoU 0.2432 → 0.9797）

### 工程补齐（本次）
- 重写 `GOSVG_TODO.md` 为当前状态单一事实来源；补全「明确不支持」清单
- 新增 CHANGELOG / CONTRIBUTING / SECURITY / examples / UPSTREAM 记录
- CI 加 `-race` / fuzz 冒烟 / arm64 交叉编译 / Go 缓存
- 新增 Example / Benchmark / golden 像素哈希回归基线（6 个确定性样例）/ cmd 测试
- `parseFloat` 拒绝 NaN/Inf、`parseLength` 单位白名单
- 核对 `internal/rasterx/stroke.go` 依赖链并写入 `UPSTREAM.md`（经编译验证不能删）

### 分支 `chore/close-gaps`（档 2 / 档 3）
- 新增 `RenderWithOptions` / `Document.RenderWithOptions` / `RenderOptions`（旧 API 行为不变）
- 新增 `Document.Diagnostics()` 观察渲染期跳过事件
- 新增 `DefaultMaxDimension=8192` 尺寸上限与 `maxNestingDepth=32` 离屏嵌套上限
- 档 2 其余项给出明确决策（module 路径保留、单位不换算、H-A 保持子集边界、tag 待合并后打）
- 档 3 逐项给出替代方案与触发条件；新增 `OPEN_SOURCE_CHECKLIST.md`
- 后续改动改走分支 + PR，main 不再直推

> 内网 Gitea 已打 `v0.1.0` annotated tag（PR #1 合并到 main 之后）；
> Release 已发至 `v0.2.2`（见下方各分支条目）；**仅推送外部 Git 仍待办**，
> 见 `OPEN_SOURCE_CHECKLIST.md`。

### 修补（分支 `fix/residual-gaps`）
- `RenderWithOptions` / `Document.RenderWithOptions` 对 nil ctx 回退 `context.Background()`，不再 panic
- `fillBackground` 改用 `color.NRGBAModel.Convert`，修正半透明预乘色偏
- `<title>` / `<desc>` / `<metadata>` / `<namedview>` 不再被误报为 unsupported 诊断
- mask / clip 离屏子上下文的取消错误回传主上下文（`sub.err` → `ctx.err`）
- 新增 `LastRenderDiagnostics()`（每次渲染重置），与累计的 `Diagnostics()` 互补
- 新增 `scripts/verify.sh`，CI 与本地手动验证同源；两个 workflow 改为调用它
- 同步合并/打 tag 之后过时的文档表述

### 分支 `feat/display-and-polish`
- **P0 正确性**：实现 `display:none`（跳过整棵子树）与 `visibility:hidden|collapse|visible`
  （继承语义，子元素可显式覆盖）；CSS 规则与内联 `style` 同样生效。此前两者被完全忽略，
  会把隐藏元素画出来（实证 16×16 全红矩形误绘 256 像素）。
- **P1 易用性**：新增 `DefaultImageDecoder()`（标准库 PNG/JPEG/GIF + data URI，零外部依赖）；
  `RenderOptions.ImageDecoder` 支持逐渲染注入；`UnsupportedError` 加 `Code` / `Err`
  与哨兵错误支持 `errors.Is`；新增 `RegisteredFonts()` / `ResetFonts()`。
- **P3/P4**：新增 `BenchmarkParse` / `BenchmarkRenderDocument` 与 `BENCHMARK.md` 实测基线；
  `scripts/verify.sh` 加 `-count=1 -cover` 与 benchmark 冒烟。
- 新增 15 项测试（display/visibility 9、默认解码 3、字体 2、错误码 1）。

### 分支 `feat/robustness-and-coverage`
- **输入规模上限**：`MaxSVGSize`（8 MiB）/ `MaxElementCount`（10 万）/ `MaxXMLDepth`（256），
  超限返回带 `Code` 的 `UnsupportedError` 与新哨兵 `ErrInputTooLarge` /
  `ErrTooManyElements` / `ErrNestingTooDeep`
- **新增 `FuzzRender` 目标**（渲染期畸形输入）与 6 个入库 corpus；`verify.sh` 增跑 20s
- **`internal/vector` 覆盖率 0% → 24.3%**：新增 even-odd 镂空 / non-zero 填充 /
  `Reset` 复用三项测试（覆盖本 fork 对上游的扩展）
- **`verify.sh` 新增 WASM 编译门禁**（`GOOS=js GOARCH=wasm go build ./...`）
- 清理 `gosvg.go` 里一段挂在 `ErrorCode` 上、内容已过时的孤立注释
- `BENCHMARK.md` 记录 pprof 分配热点，并据此明确不做 CSS 索引 / 解析缓存 / 离屏池化的理由

### 分支 `feat/limits-and-coverage`
- **CSS 解析规模上限**：`MaxCSSRules` / `MaxCSSSelectorLen` / `MaxCSSDeclLen`；
  超限为降级（丢弃 + 诊断）而非硬失败，新增 `parseCSSLimited`（`parseCSS` 保持原签名兼容）
- **并发用例**：`TestConcurrentRenderSameDocument` / `TestConcurrentParseAndRender`，
  配合 `-race` 验证同一 `Document` 可并发渲染、诊断字段锁保护有效
- **`cmd/verify` 覆盖率 0% → 62.3%**：重构为 `parseArgs` / `run` / `writePNG`，6 项测试
- **`internal/rasterx` 覆盖率 0% → 3.9%**：新增 `Matrix2D` 的 8 项测试
  （Identity / Mult / Translate / Invert 往返 / TransformVector / TFixed / SkewX / SkewY）

### 分支 `feat/realworld-compat`
全部改动由 **958 个真实世界 SVG** 的实测驱动（详见 `CORPUS_BASELINE.md`）：

- **非 UTF-8 编码声明不再硬失败**：新增 `charset.go` 与可替换的 `CharsetReader`，
  默认零依赖支持 UTF-8 / ASCII 系透传 + ISO-8859-1 逐字节展开，其余编码返回带编码名的
  明确错误。实测 Parse 失败 **67 → 0**
- **无效颜色声明按 CSS 语义忽略**：`mergeStyle` 仅在 `isValidPaint` 通过时覆盖继承值，
  `fill="000000"`（Material Symbols 真实导出写法）从全透明恢复为规范默认 black
- **`parseColor` 校验十六进制字符**：`#gggggg` 不再被 `hexv` 静默当成 0 渲染为黑色
- **`<image>` 的 data: URI 开箱可用**：未注入解码器时用标准库内置解码（不碰文件与网络）；
  文件路径 / 远程 URL 仍需显式注入，并有测试守住这条安全边界
- **新增 `cmd/corpus`**：批量渲染真实语料，输出成功率 / 全透明清单 / 失败清单 / 诊断分布，
  支持 `-json`；覆盖率 42.2%
- 真实语料整体成功率 **90.4% → 99.5%**（954/958）

### 分支 `feat/pattern-and-stroke-reuse`
- **新增 `<pattern>` 平铺填充**：`patternUnits` / `patternContentUnits` / `patternTransform` /
  `x`/`y`/`width`/`height` / tile 内 `viewBox`；tile 内支持 `<use>`、`<image>`（含 data URI）
  等任意受支持元素。实现为「离屏渲染 tile → 按 tile 位置逐块 `draw.DrawMask` →
  path alpha 掩膜」，`maxPatternTile=4096` 防止超大 tile 分配。
  `stroke="url(#pattern)"` 记诊断后跳过。真实语料效果：fnOS 面板批次 216/217 → **217/217**。
- **`strokeCapsules` 复用 Filler**：多 stroke path 场景 A/B 实测 **B/op -58%（212.5MB → 88.7MB）、
  ns/op -19%**；单 path 场景无变化。新增 `BenchmarkRenderMultiStroke` 基线。
- `<pattern>` 从「明确不支持」移入支持范围（README / 包注释 / GOSVG_TODO 三处同步）。
- 9 项 pattern 测试（含真实语料的 Figma 模式：pattern + use + base64 image）。

### 分支 `feat/clip-rule`
- **支持 `clip-rule`**（SVG 1.1 可继承属性，默认 nonzero）：`clipPath` 或其子元素上的
  `clip-rule` 映射到内部 `fill-rule`，复用既有 even-odd winding 光栅路径；每个子元素
  独立计算避免兄弟污染；CSS 里写 `clip-rule` 同样生效。6 项测试覆盖 evenodd /
  默认 nonzero / 显式 nonzero / 从 `<clipPath>` 继承 / CSS 来源 / 兄弟不泄漏。
- **实测结论（重要，负面结果如实记录）**：958 个真实 SVG 中 91 个文件含 `clip-rule`
  （128 次，全部为 `evenodd`），87 个 `clipPath` 块里只有 10 个属于「多子路径或多形状」，
  对其中 5 个候选做**修复前后逐像素对比：差异全部为 0**。原因是这些 `clipPath`
  都服务于 Figma 的 `bgblur` + `foreignObject`（已被跳过），或子路径互不重叠。
  → 本项属**语义补全**，不是「救回渲染」；golden 与语料成功率均未变化。
- **方法论修正**：上一轮用「属性出现频率」排优先级不可靠（91 文件 → 实际影响 0），
  必须做修复前后像素对比或与完整实现（cairosvg）对账才能量化真实影响。

### 分支 `fix/clip-transform-order-and-encode`
- **修复 `clip-path` / `mask` 的求值顺序**（真 bug，靠 cairosvg 对账定位）：SVG 规定二者
  定义在「引用元素的 user space」，即元素自身 `transform` 生效之后；此前 `renderNode`
  先算 clip/mask、`renderNodeBare` 才应用 transform，顺序反了。
  实测 flag-icons 阿鲁巴国旗 **IoU 0.2060 → 1.0000**，A 批平均 0.9653 → 0.9947，
  **退化文件 0 个**；`golden.sha256` 未变。
- **新增输出封装**：`RenderToWriter` / `EncodeImage`（PNG 与 JPEG，JPEG 自动合成到背景色
  避免黑底）、`RenderOptions.JPEGQuality` 与 `RenderOptions.Ctx`。
- **新增 `CAIROSVG_IOU.md`**：117 个真实语料抽样的对账报告（中位 IoU 0.9982、分布、
  按批次、低分根因分类、复现步骤）。
- 新增 4 项 transform+clip/mask 测试与 9 项 encode 测试；根包覆盖率 85.9% → **86.0%**。

### 分支 `fix/no-viewbox-aspect-ratio`
- **修复无 viewBox 时的非等比拉伸**：此前 `viewBoxMatrix` 的 `!hasVB` 分支直接
  `Scale(w/vw, h/vh)`，把 `width!=height` 的 SVG 拉扁并整体错位。改为按 width/height
  等比缩放 + `xMidYMid` 居中（与浏览器 / cairosvg 一致）；显式
  `preserveAspectRatio="none"` 仍保留拉伸。4 项测试；`golden.sha256` 未变
  （6 个基线样例全有 viewBox）。
- **cairosvg 全量对账（848 样本）**：平均 IoU **0.7870 → 0.9486**、IoU<0.5 **210 → 17**、
  D 批 0.2731 → 0.8687；**195 个文件提升 >0.05、零退化**。`resizeHorizontal` 0.0000 → 0.999。
- **撤回两个此前的误判**：
  1. D 批低分不是「亚像素 AA + 字体已定案边界」，而是无 viewBox 的非等比拉伸
     （上版依据「重叠区 RGB 差仅 3–7」在位置错开时失真）。
  2. even-odd + transform 的全量高危交叉验证显示中位 IoU 1.0000、只有 3 个 <0.5，
     **证伪该假设**，撤回改 `internal/vector` 核心的计划；`frame-icon` 是个例。
- `CAIROSVG_IOU.md` 补齐全量复测数据、根因分类与方法论校准（抽样会骗人、bbox 比 RGB 差可靠、
  假设必须全量验证）。
- 新增待查：D 批 `width="193mm"` 物理单位未换算（54 个低分文件）。

### 分支 `fix/viewport-physical-units`
- **视口尺寸按 96dpi 换算物理单位**：新增 `parseViewportLength`，根 `<svg>` 的
  `width`/`height` 支持 `px` / `mm` / `cm` / `in` / `pt` / `pc` / `q`（无单位按 px；
  `%` 与相对单位在根上无父参照，保持 1:1 近似的旧行为）。`parser.go` 的
  `d.width` / `d.height` 改走它；**元素几何仍走 `parseLengthOr` 的 1:1 近似**，
  不动既有「单位不换算」决策。
- 4 项测试：单位换算表、默认值、物理单位视口的端到端定位、有 viewBox 时视口单位
  不影响渲染字节。根包覆盖率 86.0% → **86.1%**；`golden.sha256` 未变
  （6 个基线样例均无物理单位）。
- **`scripts/verify.sh` 全套通过**（.15 / Go 1.27.3）：gofmt 空、vet 0、
  test 0、`-race` 0、arm64 / wasm 交叉编译 0、bench 冒烟、
  `FuzzParse` 30s + `FuzzRender` 20s 无 crash → `ALL GREEN`。
- **全量对账的诚实结论（收益很小）**：848 样本平均 IoU **0.9486 → 0.9507**，
  D 批低分 **54 → 52**，退化 **0**，只救回两个 bamboo 文件
  （`0.0000 → 0.9334` / `0.8604`）。即物理单位换算**不是**剩余 52 个低分的原因。
- **剩余 52 个定性**：50 个为渲染器抗锯齿 / 字体栅格化的像素级边界差异
  （bbox 与不透明像素数都近似，IoU 0.70–0.77 已非几何错误）；
  2 个为填充差异（`toolbarButton-editorHighlight` 0.40、`altText_spinner` 0.45）。
  判定为**已定案边界，不值得继续改**，详见 `CAIROSVG_IOU.md`。

### 分支 `fix/viewport-relative-units-and-docs`
- **修 v0.2.2 新引入的相对单位回归**：`parseViewportLength` 原先把 `width="100em"` 之类的
  相对单位交给 `parseFloat` 判非法、返回默认 0（视口尺寸归零），与它自己承诺的
  「相对单位保持 1:1 近似旧行为」相矛盾。改为先剥离后缀再按单位分派：物理单位换算、
  相对单位（`em/rem/ex/ch/vw/vh/vmin/vmax/%`）按数值 1:1 近似、未知后缀判非法；
  顺带修好注释里「按后缀长度降序匹配」与代码不符的误导。
- 测试从 12 例扩到 27 例（补大小写、科学计数法、负数、空白、8 种相对单位、非法后缀），
  新增端到端 `TestViewportRelativeUnitsRender`；**`golden.sha256` 未变**。
- **文档一致性批量补齐**（本轮盘点出 10 处自相矛盾 / 过时）：包注释与 `GOSVG_TODO.md`
  里「渲染期跳过事件没有诊断通道」→ 已实现 `Diagnostics()` / `LastRenderDiagnostics()`；
  `<image>` 三处仍写「必须注入解码器」→ 已有开箱可用的 `DefaultImageDecoder()`；
  覆盖率 85.8% → **86.2%**（本轮实测；其余包 62.3% / 42.2% / 16.7% / 24.3% / 3.9% 复核无误）；
  CHANGELOG / CHECKLIST / GAP TODO 里
  「Release 待办」→ 已发至 v0.2.2；`ResetFonts()` 隔离、分支保护 `protected=True`、
  H-A 表「视口单位例外」逐项同步；`CAIROSVG_IOU.md` 修正 1 的旧结论第 2/4 条
  标注「已被修正 2/3 推翻」，避免被推翻的判断继续误导后续排查。
- **推翻上一版对 2 个低分文件的定性**：PNG 逐像素复核显示二者是**整块缺失的真 bug**
  （gosvg 只多画 5px，cairosvg 多画 12220 / 18661 px），不是「填充差异 / AA 边界」。
  只读定位到 `@media` 嵌套 CSS 规则被误应用、even-odd 多子路径被挖空两个根因假设；
  **改动会波及渲染字节与全量 IoU 表，按纪律列为需用户点头项，未动代码**。

### 分支 `feat/evenodd-path-conversion`
- **修 even-odd 多子路径的系统性渲染缺陷**（用户授权「按你的建议来」后落地修正 6 的 PoC）：
  新增 `evenodd_convert.go`，在解析层把 `fill-rule="evenodd"` 的多子路径按嵌套深度
  调整为方向交替、改走 nonzero；`renderShape` 只把转换后的 Path 用于填充，
  **描边与 marker 仍用原 path**。`internal/rasterx` / `internal/vector` 光栅器核心零改动。
- 保守触发：`d` 不含 `A/a/T/t`、≥2 子路径、不在 clipPath 内、无自交 / 部分重叠 /
  零面积子路径、采样点数 ≤ 512；任一不满足即回退原 even-odd，不引入新的错误填充。
  `inClipPath` 标志由 `clip.go` / `mask.go` / `pattern.go` 的离屏 context 传递，
  保证 clip-rule 语义不受影响。
- `evenodd_convert_test.go` 10 项测试覆盖同向嵌套、三层 winding、独立子路径、
  三次贝塞尔控制点倒序、gradient、clipPath 跳过、A/T 跳过，以及自交 / 部分重叠两类拒绝用例。
- **`verify.sh` 全套 ALL GREEN**；`golden.sha256` 核验后**无需重生成**（6 个基线样例均无
  `fill-rule` 声明），不是漏检。
- **全量生效验证**（干净基线 vs 落地后各渲染 850 个 SVG，848 个与 cairosvg 对账）：
  **30 个文件像素变化、30 提升 / 0 退化 / 818 不变**，平均 IoU
  **0.949874 → 0.961272（+0.011397）**，`IoU<0.5` 从 **15 → 4**，
  11 个目标文件平均 **0.2560 → 0.9927**（`B__frame-icon` 0.0602 → 1.0000）。
  剩余 4 个低分文件像素零变化，均属 clip-path / `@media` 等非 even-odd 问题。
  详见 `CAIROSVG_IOU.md` 修正 7。
- 未打 tag、未发 release，发版仍需单独确认。
