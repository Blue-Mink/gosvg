# Changelog

所有 notable 变更记录在此。提交信息沿用 Conventional Commits
（`feat:` / `fix:` / `docs:` / `chore:` / `test:`）。

## [Unreleased]

- **文档状态收口（PR #23，纯文档、零渲染改动）**：v0.2.4 发布后 6 个 `.md` 的版本
  状态收口；main 前移 1 个合并提交，`v0.2.4` tag 未移动。
- **文档一致性收口（本次，纯文档 / 脚本、零渲染改动）**：修正全仓体检发现的版本
  状态倒挂（`[Unreleased]` 误写「暂无改动」、「`v0.2.4` = main」失效）、悬空的
  「静默分支清单」引用、6/7 个 golden 样例、测试函数计数口径（215 历史粗计数 /
  231 = 228 `Test` + 3 `Example`）、`altText_spinner` 已修却仍写「待查项」、
  H-A 表自相矛盾等；`verify.sh` 新增
  `check_docs.sh` 门禁与 `audit_docs.py` 深度审计（可选、不阻塞）。`golden.sha256`、
  渲染代码、全量 IoU 基线一律未动。
- **even-odd 等价转换覆盖补全（修正 10，本次；渲染行为变更）**：PR #17 的
  `evenOddPathEligible` 有两道过度保守的门槛，把仍受光栅器 even-odd 累积偏差
  影响的 shape 放过了：① `moves>=2` 让**单子路径**直接跳过（源码注释假设
  「单子路径 evenodd 与 nonzero 等价，无需转换」，几何上成立，但光栅器实现
  对这类 shape 同样整块少画，实测证伪）；② 按**原始 d 串**扫描 `A/a/T/t`
  拒绝，而 `parsePathData` 早已把弧线 / 平滑指令归一化为 `CubicTo`/`QuadTo`
  （都是 `eoSamplePoly` 支持的段）。落地：单子路径且不自交时 `renderShape`
  直接用 nonzero 填充（新增 `eoSingleNonSelfIntersecting`，自交者保留原语义）；
  弧线门槛改为只看归一化后的 `rasterx.Path`。`internal/rasterx` /
  `internal/vector` 零改动。全量 850 按二值 alpha mask 逐像素：**10 文件变化 / 10 提升 / 0 退化 /
  838 不变**（848 可比样本），平均 IoU `0.963860 → 0.965544`（Δ +0.001684），
  `IoU<0.8` 55 → 53、`IoU<0.5` 仍 1（wacom）。完整 RGBA 字节层面另有
  47 个文件仅 RGB 变化（alpha 逐位相同），参照图 RGB 平均差 `88.73 → 15.02`、
  43 改善 / 4 局部变差（均 IoU=1.0000，已知局限已入档）—— alpha IoU 口径
  对这类颜色纠正不敏感。golden 7 样例完整 RGBA 逐像素零变化，
  `golden.sha256` 未动。新增 4 个回归测试，其中渲染级测试在基线上 FAIL / 修复版 PASS。
  详见 `CAIROSVG_IOU.md` 修正 10。

- **源码注释漏网订正 + 门禁覆盖扩展（纯注释 / 脚本 / 文档，零渲染改动）**：
  PR #24 的深度审计只扫 `.md`，遗漏 `cmd/corpus/main.go` 包注释里把固定样例数
  写死为旧值（v0.2.3 纳入 `s60s` 后已扩为 7 个）。该行改为不写死数量、引用
  `testdata/*.svg` 自动枚举；`check_docs.sh` 新增硬检查「源码注释不得写死固定
  样例数」，`audit_docs.py` 新增 F 段把扫描范围扩到 `.go` / `.sh` / `.py` 注释；
  `CONTRIBUTING.md` / `GOSVG_TODO.md` 同步门禁覆盖范围。`golden.sha256`、
  渲染代码、全量 IoU 基线一律未动。

## [v0.2.4] — 2026-11-11

`v0.2.3..v0.2.4` 共 **15 个提交**（PR #19–#22），tag 由 Gitea Release API 自建，
tag `v0.2.4` → `67564f4`（**发布当时** = main；此后 main 经 PR #23 等提交持续
前移，tag 未移动，两者不再等同；当前 main 以 `git rev-parse HEAD` 为准）。
全量对账数字保持不变：平均 alpha IoU
**`0.963860`** / `IoU<0.5` **1** 个 / 848 个可对比样本（`alpha > 0`、256×256）。
本版唯一行为变更是 `DefaultImageDecoder()` 收紧为仅 `data:` URI（见下）。

> **版本状态注记（发布后补）**：v0.2.4 的 Gitea Release 附注里写「`CHANGELOG.md` 保留
> `[Unreleased]` 标题（沿用 v0.2.0–v0.2.3 惯例）」，那是**发布当时**的快照；本段已在本
> 次发布后收口中转正为 `[v0.2.4]`，附注原句不再代表当前状态。Release 附注作为已发布
> 对象保留原文不改，当前版本状态以本文件为准。

### 发版后全量审计修复（v0.2.3 之后）
- **文档收口**：`CHANGELOG` / `README` / `CONTRIBUTING` / `SECURITY` /
  `CAIROSVG_IOU.md` 补 v0.2.3 发布状态与 848 样本全量对账结论（此前仍写「未发 release」，
  README 停在已废弃的 117 抽样结论）；两处 CI workflow 补 `push.branches` / `push.tags` /
  `workflow_dispatch` 触发、Go 固定 1.27.3、去掉硬编码 GOPROXY（改由 `verify.sh` 兜底）。
- **CSS 解析**：`css.go` 的 `parseCSSLimited` 加引号感知的选择器 / 声明扫描，修
  `content:"{"` / `content:"}"` 里的括号被误当规则边界；新增 `css_quote_test.go` 6 例。
- **golden 稳定性**：`font_test.go` 用 `t.Cleanup(resetFonts)` 复位包级 `fontRegistry`，
  修 `TestRenderUsesSystemFont` 污染导致的 flaky；`golden_test.go` 改为自动枚举
  `testdata/*.svg` 并纳入带 `<text>` 的 `s60s`，基线 **6 → 7 个样例**，
  `-shuffle=on` 全过；`golden.sha256` 仅新增 `s60s` 一行，已有样例哈希未变。
- **门禁**：`go mod tidy` 修正 `golang.org/x/sys` 误标的 `// indirect`；
  `scripts/verify.sh` 加 `go mod verify` / `go mod tidy -diff`，fuzz 冒烟从 2 个目标
  扩为 6 个全覆盖；`.gitignore` 精确化；`internal/vector/UPSTREAM.md` 更正字体隔离结论。
- **验证**：`sh ./scripts/verify.sh` ALL GREEN（当时根包 86.5%；本次补 fuzz
  引号种子后 86.7%）；全量 850 语料 0 命中新 CSS 分支，平均 IoU `0.963860` 仍有效。

### 文档二轮收口（本次）
- `GOSVG_TODO.md` 的 golden 段落同步为 7 个样例（此前仍写「6 个 / s60s 暂不纳入」，
  与 `GOSVG_GAP_TODO.md` 冲突）、补 `-update` 警示、fuzz 数量与冒烟命令对齐。
- `CAIROSVG_IOU.md` 修正 7 的「6 个确定性样例」加时间注记（后续审计已扩为 7 个）。
- `OPEN_SOURCE_CHECKLIST.md` 勾选两项：全历史敏感串扫描（101 提交 / 19337 diff 行
  0 命中）、许可证核对（3 份 LICENSE 齐全）。其余项依赖推外部 Git，按当前约束保持未勾。
- `FuzzStyle` 补 3 个 CSS 引号种子（`{` / `}` / `;`），让变异器命中引号感知分支。

### 已知边界（v0.2.4 发布时）
- `ImageDecoder` / `CharsetReader` 为包级可变 hook，运行时替换需调用方自行同步；
  **IoU 对账工具已由本版 `cmd/iou` 入仓关闭**（此前记为缺口）；
  内网 Gitea CI runner 数仍为 0，workflow 语法变更未经服务端实跑。

### parseStyle 引号/括号感知（PR #20）

- **修复**：`style.go` 的 `parseStyle` 不再用裸 `strings.Split(s, ";")` 切分 style 属性，
  改由新增的 `splitStyleDecls` 按**顶层分号**切分 —— 引号内字符串复用 `css.go` 的
  `skipCSSQuoted` 跳过（含 `\'` 转义），括号用深度计数跳过 `url(...)` 等函数式记号
  内部的分号。
- **修掉的两种坏值**：`style="content:'a;b';fill:red"` 的 `content` 值曾被截成 `'a`
  且尾随 `b'` 变非法声明被丢弃；`style="fill:url(data:image/png;base64,AA)"` 的值曾被
  截成 `url(data:image/png`，图片引用失效。
- **测试**：新增 `style_quote_test.go` 覆盖引号 / 双引号 / data URI / 转义引号 /
  引号内花括号 7 个场景，另有一例确认基础语义（key 小写、缺冒号与空值静默跳过）不变。
  测试有效性：把 `style.go` 回退到修复前版本后，4 个子用例 FAIL（精确复现缺陷）。
- **验证**：`sh ./scripts/verify.sh` ALL GREEN（RC=0，根包覆盖率 86.7% → 86.8%）；
  **全量 850 语料唯一变量对比**（HEAD 二进制 vs 修复版二进制渲染同一批输入）
  **850/850 像素完全一致** —— 现有语料 0 命中新分支，故渲染输出零变化。
- **基线**：`testdata/golden.sha256` 无需更新（7 个样例哈希不变）；未触碰任何光栅器
  与渲染路径文件（仅 `style.go` + 新测试文件）。
- **状态**：经 **PR #20** 合入 main，**已随 v0.2.4 发布**。
- 上文「已知边界」中的 `parseStyle` 引号缺口**已由本条修复**，故从该段落移出。

### IoU 对账工具入仓（`cmd/iou`）

- **背景**：全量 848 样本的平均 IoU / `IoU<0.5` 计数等数字此前只能靠仓库外一次性
  脚本复现（`GOSVG_GAP_TODO.md` 档 3 C1 记为缺口）。新增 `cmd/iou` 把口径固化进仓库。
- **能力**：按同名 stem 配对参照图与 SVG，算 alpha mask 的 `交/并`；`-alpha 0`（默认，
  与全量数字同口径）与 `-alpha 8`（`CAIROSVG_IOU.md` 正文口径）可切换，两套数字
  **不可直接横比**；`-min` 筛低分、`-json` 导出完整结果、`-baseline` 统计
  提升 / 退化 / 不变、`-out` 落盘渲染 PNG、`-size` 指定边长。
- **验证**：`cmd/iou` 6 项单元测试全过；工具在当前 main 上实测复算得
  `average IoU = 0.963860` / `IoU<0.5 = 1` / 848 个可比样本，与 `[v0.2.3]` 记录
  **逐位一致**；`-alpha 8` 得 `0.964787`；低分清单唯一命中
  `D__wacom-intuos-pro-2-s` 的 `0.4397`，亦与既有记录一致。
- **测试有效性**：`TestRenderSVG` 曾抓到 `renderSVG` 未自建输出目录的真 bug，已修。
- **状态**：**已随 v0.2.4 发布**。

### `<image>` 解码器安全收紧（行为变更）

- **风险**：此前 `DefaultImageDecoder()` 对非 `data:` 的 `href` 直接 `os.Open(href)`。
  服务端渲染不可信 SVG 时，`<image href="/path/to/secret">` 可读任意本地文件。
  （库默认路径一直是安全的：未注入解码器时仅 `data:` URI 内置解码、其余静默跳过；
  风险只出现在调用方主动注入默认解码器之后。）
- **收紧**：`DefaultImageDecoder()` 改为**仅解 `data:` URI**，等价于内置
  `decodeDataURIOnly`，不碰文件系统与网络。
- **显式开关**：新增 `LocalFileImageDecoder(allowedRoots ...string)`。传入白名单根目录
  时，路径先解析 `..` 与符号链接再校验是否越界，越界返回错误、不打开文件；远程 URL
  一律拒绝。不传 `allowedRoots` 则允许任意本地路径（等价旧行为，**仅供完全可信输入**）。
- **迁移**：依赖旧「注入默认解码器即可读本地文件」的调用方需改用
  `LocalFileImageDecoder(...)`；只处理 `data:` URI 的代码无需改动。
- **测试**：新增 `imagedecode_security_test.go` 4 项（拒绝本地文件 / 白名单内放行、
  白名单外与 `../` 穿越拒绝 / 无白名单旧行为 / 端到端不产生像素）。
  **测试有效性**：回退到收紧前实现后，独立探针 FAIL（确实读到了本地文件），
  新实现 PASS —— 确认不是空转。
- **文档**：`SECURITY.md` / `README.md` / `GOSVG_TODO.md` / `gosvg.go` / `image.go`
  同步；`image.go` 的 `decodeDataURIOnly` 边界说明不变。
- **渲染影响**：850 语料中 `<image href>` 仅 9 处且全为 `data:` URI，
  渲染输出零变化；`testdata/golden.sha256` 无需更新。
- **状态**：**已随 v0.2.4 发布**。

### 最后一个低分样本的定性纠错（纯文档）

- 全量 848 样本中唯一 `IoU<0.5` 的 `D__wacom-intuos-pro-2-s`（0.4397），
  `CAIROSVG_IOU.md` 修正 8 原记为「9 个 `<text>` 的字体度量 / AA 边界」，
  **该定性被消融实验推翻**。
- 六组只改 SVG 的 A/B：改 `font-size`、删掉全部 9 个 `<text>`、给 text 显式
  字号，gosvg 渲染**逐像素完全不变**（537 px）→ 文字根本没画；去掉根
  `stroke:#7f7f7f` 后像素 537→0 → gosvg 画的 537 px 全是 stroke 线条；
  给 text 补 `fill:black` 后 IoU 仅 0.4397→0.4439。
- **真因**：根 `<svg>` 指定灰色描边 + `fill:none`，cairosvg 画了文字轮廓
  （缺失的 334 px RGB ≈ `[121,121,121]` 正是 `#7f7f7f`）；gosvg 的 `text.go`
  完全不处理文字描边，且 `fill:none` 按语义跳过填充 → 9 个文字全不画。
- **定案仍为不修**（要修需实现文字轮廓描边，牵涉字体轮廓转路径、`.25` 亚像素
  线宽与 `fill:none` 语义，收益仅一个文件、风险覆盖整个 text 路径）。
  详见 `CAIROSVG_IOU.md` 修正 9。零代码改动、零渲染影响。

## [v0.2.3] — 2026-11-10

`v0.2.2..v0.2.3` 共 12 个提交（PR #14–#18）。tag 由 Gitea Release API 自建，
未先手动 push tag（避开 `is_tag` 占位 UNIQUE 冲突）；`draft=false` / `prerelease=false`，
tag `v0.2.3` → `3d9072f`。下方各分支条目为明细，净结果：

### 净结果（v0.2.2 → v0.2.3）
- **even-odd 系统性缺陷修复**（PR #17，修正 7）：解析层把单条 path 内多 `Z` 子路径按
  交替方向等价转换为 non-zero，零光栅器改动；自交 / 部分重叠自动拒绝转换。全量 848
  样本 **30 提升 / 0 退化 / 818 不变**，平均 IoU `0.949874 → 0.961272`（+0.011397），
  `IoU<0.5` **15 → 4**；`golden.sha256` 核验无需重生成。
- **修正 4 假设证伪并归档**（PR #13 关闭 / #14 合并，修正 5）：`render.go` / `gradient.go`
  独立 `Filler` 实例经 848 样本逐像素对账为 **no-op**（像素 sha `915deee870f66710`
  完全一致）。方法论补「文档结论必须与可执行验证同源」。
- **影响账纠错 + PoC**（PR #15 / #16，修正 6）：15 个低分里 **11 个**同时声明
  `fill-rule="evenodd"` + `clip-rule="evenodd"`（富集 27 倍）；A/B 实验证明只改
  `fill-rule` 有效。SVG 预处理 PoC 让 11 文件 `0.2533 → 0.9924`、全量 61 个改动文件
  30 提升 / 31 不变 / 0 退化；拆 path 方案证伪（`0.7817`，PR #15 初稿分母
  `0.7107` 已由 #16 更正）。
- **剩余 4 个低分的结构性修复**（PR #18，修正 8）：`css.go` at-rule 深度扫描、
  `render.go` 的 `skipNodeTransform` 泄漏、`image.go` 的 `srcRect` 取源切片；3 个回归
  测试在基线上全 FAIL / 修复版全 PASS；全量 848 平均 IoU `0.961272 → 0.963860`、
  `IoU<0.5` **4 → 1**；1 处已知退化 −0.0223 已入档。`verify.sh` ALL GREEN
  （根包覆盖率 86.5%）。

### Added
- 纯 Go SVG 子集渲染器：`Render` / `RenderTo` / `Parse` / `Document` 主 API
- fill/stroke（胶囊体并集精确轮廓）、even-odd winding、gradient（fill + stroke）
- mask（luminance，units/region/transform）、clip-path、tspan、marker、`<image>`（回调注入）
- CSS `<style>` + class/id/类型选择器、hsl()/现代颜色、dasharray/dashoffset
- preserveAspectRatio 9 种 align + meet/slice/none、MeasureText/MeasureTextEx
- 可选系统字体目录 `AddFontDir`
- 6 个 fuzz 目标、215 个测试函数（**当时粗计数、口径未细分**；v0.2.4 实测为
  **228 个 `Test` + 3 个 `Example` = 231**，命令 `go test -list '^(Test|Example)' ./...`）、
  Gitea + GitHub 双 CI 配置
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
- 新增 Example / Benchmark / golden 像素哈希回归基线（自动枚举 `testdata/*.svg`，
  当前 7 个样例）/ cmd 测试
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
> Release 当时已发至 `v0.2.3`（现最新为 `v0.2.4`，见文首）；**仅推送外部 Git 仍待办**，
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
- **`<image>` 的 data: URI 开箱可用**：未注入解码器时用标准库内置解码（仅限 `data:` URI，不碰文件与网络）；
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
  ⚠️ **此定性已被修正 6 / 7 推翻**：15 个 `IoU<0.5` 中 11 个是 even-odd 系统性缺陷
  （不是 AA 边界），已由 PR #17 / #18 修复。本段仅作为当时的判断留存，勿据此继续排查。

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
  覆盖率 85.8% → **86.2%**（当时实测；当时最新 86.7%，v0.2.4 为 **87.0%**；其余包 62.3% / 42.2% / 16.7% / 24.3% / 3.9% 复核无误）；
  CHANGELOG / CHECKLIST / GAP TODO 里
  「Release 待办」→ 已发至 v0.2.3；`ResetFonts()` 隔离、分支保护 `protected=True`、
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
- 本行为当时快照；v0.2.3 已于 2026-11-10 发布（见上方「净结果」）。
