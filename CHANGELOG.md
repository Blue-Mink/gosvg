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
> Release 与推送外部 Git 仍待办，见 `OPEN_SOURCE_CHECKLIST.md`。

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
