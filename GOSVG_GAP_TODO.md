# GOSVG 缺口清单与补齐进度

> 由 2026-11-07 起多轮审阅合并而成。状态：**档 1 已补齐、档 2 已处置、档 3 已给出
> 触发条件**；PR #1 已合并到 main，内网 Gitea 已打 `v0.1.0`（当时；最新 release 为
> **v0.3.0**，见 `CHANGELOG.md`）。本文件记录决策与待办，不再是「待拍板」清单。
> 本文按轮次累积，各轮「当前」「仍是」「仍为」均为该轮当时的快照，不代表最新状态；
> 已被后续实测推翻的结论以段后 ⚠️ 注记标明。

## 档 1 — 已补齐（零决策、不破坏 API、不改渲染语义）

| 项 | 落地 |
|---|---|
| A1 `GOSVG_TODO.md` 过时自相矛盾 | 重写为「当前范围 / 明确不支持 / 已知边界 / 质量基线 / 变更记录」，删除 4 处过时结论 |
| A2 「明确不支持」清单严重不全 | README + 包注释补全 13 类静默丢弃特性 |
| A3 文档措辞夸大实现 | `style.go` + TODO + README 统一为「单位 1:1 近似」 |
| A4 `UnsupportedError` 注释语义错位 | 修正为「调用方输入错误」，明确特性跳过不返回此错误 |
| B2/B3 错误类型盘点 | 包注释与 `SECURITY.md` 写清 4 类错误与诊断缺口 |
| C1 CI 缺 fuzz 门禁 | 双 CI 加 `-fuzz=FuzzParse -fuzztime=30s` 冒烟 |
| C2 `.gitignore` 忽略 `*.png` | 放行 `testdata/golden/*.png`；回归基线改用 `golden.sha256` 文本 |
| C3 CI 缺 race/静态检查 | 加 `-race`、`go vet`、Go 模块缓存 |
| C4 无 ARM64 验证 | 加 `GOARCH=arm64 GOOS=linux go build ./...`（真机测试仍需 runner） |
| C5 无 Example / Benchmark | 新增 `example_test.go`（3 个 Example + `BenchmarkRender`） |
| C6 `internal/*` 与 `cmd/*` 无测试 | 新增 `golden_test.go`、`cmd/render/main_test.go`；`cmd/verify` 收拾为可用工具 |
| D1 `internal/rasterx/stroke.go` 疑似 677 行死代码 | **编译验证推翻该判断**：`dash.go`/`shapes.go` 依赖其中的 `Stroker`/`CapFunc`/`GapFunc`/`JoinMode`，已保留；把依赖结论写入 `UPSTREAM.md` |
| D2 fork 无上游同步记录 | `internal/rasterx/UPSTREAM.md`、`internal/vector/UPSTREAM.md` |
| E2 无 CHANGELOG / CONTRIBUTING / SECURITY / examples | 全部补齐 |
| G3 NaN/Inf 无防护 | `parseFloat` 拒绝 NaN / ±Inf |
| G4 `parseLength` 剥离任意字母 | 改单位白名单，`12abc` 判非法 |
| I1 无性能基线 | `BenchmarkRender` |

## 本轮验证新发现（未修，如实记录）

- **`internal/rasterx/stroke.go` 不是死代码**：删除后 `dash.go` / `shapes.go` 编译失败
  （依赖 `Stroker` / `CapFunc` / `GapFunc` / `JoinMode` 及 Dasher 字段）。原「677 行死代码」
  判断错误，文件已恢复，依赖结论写入 `internal/rasterx/UPSTREAM.md`。
- **`AddFontDir` 全局状态污染测试**（**已真正修复，2026-11-10**）：`font_test.go` 的
  `TestRenderUsesSystemFont` 注册系统字体后污染包级 `fontRegistry`，实测 `s60s.svg`
  单独跑 golden 与跑完整 `go test ./...` 的像素哈希不同（`bb150194…` vs `dc09dbc6…`）。
  - **此前本条写「隔离能力已落地 / 档 2 F4 已完成」是错误的**：`font_extra_test.go`
    只验证了 `ResetFonts()` 幂等，测试本体并未在结束时复位，flaky 一直存在。
  - **本次修复**：`TestRenderUsesSystemFont` 加 `t.Cleanup(resetFonts)`；`golden_test.go`
    改为自动枚举 `testdata/*.svg`、纳入 `s60s`（现 7 个样例）、加空图 / 全透明断言。
    单独跑、全量跑两次、`-shuffle=on` 全过，`s60s` 哈希稳定为 `bb150194…`。
    档 2 F4 至此才真正关闭。

## 档 2 — 已处置（分支 `chore/close-gaps`）

| 项 | 处置结论 | 落地 |
|---|---|---|
| A5/E1 module 路径 | **保留** `github.com/Blue-Mink/gosvg` | 改 module 路径是破坏性变更，且用户约束「不推外部 Git」；README 已补 `replace` / `GOPRIVATE` 说明 |
| E1 开源收尾 | **不推，留清单** | 新增 `OPEN_SOURCE_CHECKLIST.md`，逐项列出推送外部 Git 前要做的检查 |
| A3/G4 单位语义 | **不真换算**，保持 1:1 近似 | 真换算会改变渲染字节与既有 IoU；决策已写进 `GOSVG_TODO.md` 与 `style.go` 注释 |
| B1 Diagnostics 形态 | 选 `Document.Diagnostics() []string` | 纯加法、不改签名、累计去重、并发安全；覆盖主要跳过点 |
| F1/F2 API 扩展 | 新增 `RenderWithOptions` + `RenderOptions`，旧 API 不变 | `MaxDimension` / `Background` / context 取消；**超采样不做**（会改渲染管线与 golden 基线） |
| G1/G2 资源上限 | `DefaultMaxDimension=8192`、`maxNestingDepth=32` | 超限返回 `UnsupportedError`；离屏嵌套逐层 +1，并在 3 个子 `renderContext` 里继承 ctx/opts/nesting/err |
| H-A 子集内语义 | **保持为明确子集边界，不实现** | 逐项理由与复杂度见下表 |
| E3 版本发布 | **已打 `v0.1.0`** | PR #1 合并到 main 后打的 annotated tag；Release 待建（见档 3） |
| 分支策略 | **已改为分支流** | 新建 `chore/close-gaps`，后续改动走分支 + PR，main 不再直推 |

### H-A 逐项决策（三项不实现 / 四项已推翻改为实现）

| 特性 | 决策 | 理由 |
|---|---|---|
| `clip-rule` | **已推翻原决策，改为实现** | 实测 91 个文件 / 128 次使用；实现只需把它映射到既有 even-odd 路径，成本远低于原估计。**但实测真实影响为 0**（见下） |
| `vector-effect="non-scaling-stroke"` | **已推翻原决策，改为实现** | 描边管线调整后可支持：设备宽度恒等于 stroke-width（非等比缩放沿用 matrixScale 的 x/y 平均近似）。已加像素级回归测试。 |
| `<image>` 的 `preserveAspectRatio` | **已推翻原决策，改为实现** | 复用 `parsePreserveAspectRatio`：meet 等比留白 / slice 覆盖裁剪 / none 拉伸；目标与 src 宽高比在容差内一致时退化为 none（消除取整噪声）。已加像素测试（meet=800 / slice=1600 / none=1600）与 850 语料 A/B（9 个 image 文件逐字节一致）。 |
| `marker orient="auto-start-reverse"` | **已推翻原决策，改为实现** | `renderMarker` 加 `atStart` 参数，`orient="auto-start-reverse"` 时起点角度 +π、mid/end 与 auto 一致。已加像素级回归测试（起点重心偏移 7.5px、end 不受影响）。 |
| CSS `inherit` | 不实现 | CSS 级联层需引入「继承值」概念，超出简单选择器子集 |
| `use` 的 `width/height` | 不实现 | 需按 symbol/view 语义做视口裁剪，牵动 `renderUse` |
| 长度单位真换算 / `%` 基准（**元素几何**） | 不实现 | 会改变现有全部含单位样例的渲染字节与 IoU 表；**根 `<svg>` 视口尺寸是例外**，已按 96dpi 换算（v0.2.2） |

> 共同理由：上述标「不实现」的三项都属于**扩大子集**，会改变渲染结果、要求重跑
> cairosvg 对账与 golden 基线，已超出「零决策、不破坏语义」的补齐范围。需要哪个
> 请单独点名，单独做。`clip-rule` 是例外：已在档 2 实测后推翻原决策并实现。

## 档 3 — 处置结论与触发条件

| 项 | 当前处置 | 触发条件 / 具体做法 |
|---|---|---|
| E1 GitHub 侧确认与推送 | `OPEN_SOURCE_CHECKLIST.md` 留清单 | 用户解除「不推外部 Git」后：按清单核 → 建外部仓 → 加 remote → push main + tag |
| C4 ARM64 真机测试 | CI 已有 `GOARCH=arm64 go build ./...` 交叉编译门禁 | 有 arm64 runner 时加 job `runs-on: [self-hosted, linux, arm64]` + `go test ./...`；或用 `qemu-user-static` 跑交叉编译出的测试二进制。当前 .15 只有 x86_64，**不引入外部 action** 以保持 CI 稳定 |
| C1 CI 内 cairosvg 对账 | `testdata/golden.sha256` 做 gosvg 自身像素哈希回归；**`cmd/iou` 已入仓**（可复算 IoU） | 愿意加 CI 依赖时：`apt install python3-cairosvg` 生成参照图 → `go run ./cmd/iou -ref ... -svg ...` 当门禁。当前 .15 只有 x86_64 且 runner 数为 0，尚未接进 CI |
| D4 低版本 Go 兼容 | **决策：锁定 Go 1.27，不追求低版本兼容** | `go.mod` 声明 `go 1.27.3`；若未来要降级，下调 go 指令并给 CI 加版本矩阵（`1.26.x` 等）验证 |
| H-B（filter/pattern/foreignObject/动画） | **明确子集边界，不补** | 已在 README / 包注释 / `GOSVG_TODO.md` 三处声明，无需动作 |

> 档 3 的共同点：不是「做不到」，而是**受用户约束或缺少外部条件**。每项都给了替代
> 方案与触发条件，条件满足时可直接按表推进，不需要重新讨论范围。

## 已完成的两批分支

- `chore/close-gaps`（**已合并**，PR #1）：档 2 决策落地 + 档 3 处置结论 + 改走分支流。
  合并后 main = `6a84a21`，随后打 `v0.1.0`。
- 第一批 6 个 commit 曾直推 main（`e7502b0`）；历史已保留，未 force push 改写。

## 本轮（分支 `fix/residual-gaps`）

合并后盘点出的残余缺口，按 P0 → P1 → P2 修补：

- **P0-2** nil ctx panic：两处入口回退 `context.Background()`（原 `ctx.Err()` 在 nil 上必崩）。
- **P1** 代码边界：元数据元素白名单、`fillBackground` 颜色转换、离屏 `sub.err` 回传、
  `LastRenderDiagnostics()`、补 5 项回归测试。
- **P2** 文档漂移：CHANGELOG / GAP TODO / CHECKLIST / SECURITY / TODO / README 里
  「尚无 tag」「档 2 待拍板」「Diagnostics 计划」「详见 options.go」等过时表述全部同步。
- **P0-1** CI 门禁：新增 `scripts/verify.sh` 让验证可复现、两个 workflow 改调它；
  但 **Gitea Actions runner 数仍为 0、GitHub 出站不通**，CI 尚未真正执行，
  当前以 `sh ./scripts/verify.sh` 手动实跑为准。
  （2026-11-10 已修正两份 workflow：固定 Go 1.27.3、显式 `tags: ['v*']` 与
  `workflow_dispatch`、去掉 `GOPROXY` 覆盖、`verify.sh` 覆盖全部 6 个 fuzz 目标；
  注册 act_runner 需动 .15 上的服务，在用户明确授权前不做。）
- **P0-3** 分支保护：**已开启**（实测 `protected=True`），PR #2 起全部走分支 + PR 合并。

## 仍需点名的（P3，未动）

- Release **已发至 `v0.2.4`**，此条关闭；删除已合并的 `chore/close-gaps` 分支仍需点名
- 修 Gitea SQLite 锁（上轮方案 B：降频刷库的 Actions workflow）
- 注册 act_runner（当前环境无二进制、GitHub 出站不通，需你提供/授权）
- 档 2 判为「保持子集边界」的 H-A 七项，想要哪个单独点名

## 本轮（分支 `feat/display-and-polish`）

按「优化完善盘点」的建议顺序推进：

- **P0** `display` / `visibility` 正确性缺口已修（此前实证 256 像素误绘）。
- **P1** 默认解码 / 逐渲染 `ImageDecoder` / 错误类型化 / 字体枚举与 Reset 已实现。
- **P3/P4** 性能基线（`BENCHMARK.md`）与覆盖率门槛（`verify.sh`）已落地。
  实测修正了上一轮的判断：`Parse` 仅 3.5 µs、复用 `Document` 无实质收益，
  「解析缓存」不是优化方向；瓶颈在渲染/光栅化（292k allocs、5.37 MB/op）。
- 本轮**未做**、留待按兴趣点名：输入字节/元素/XML 深度上限、`FuzzRender` 与 corpus 入库、
  CSS 选择器索引、离屏位图池化、`internal/*` 覆盖率、WASM 编译验证、fnOS FPK 集成、
  Lucide / circle-flags 真实语料对账。
- 环境阻塞不变：Gitea Actions runner 仍为 0、GitHub 出站不通，门禁以手动
  `sh ./scripts/verify.sh` 实跑为准。

## 本轮（分支 `feat/robustness-and-coverage`）

- **输入规模上限**已落地（字节 / 元素数 / XML 深度），补齐了「解析阶段无任何限制」的缺口。
- **`FuzzRender`** 已加（此前 5 个目标全在解析/数值层，渲染期无 fuzz），corpus 入库 6 个；
  `verify.sh` 现跑 FuzzParse 30s + FuzzRender 20s。
- **`internal/vector` 覆盖率 0% → 24.3%**（even-odd 是本 fork 对上游的扩展，此前零测试）。
  ⚠️ **状态更新**：后续又补 `rasterizer_extra_test.go`（12 个 Test），**24.3% → 83.4%**，
  `Draw` 33.3% → 100%；见「本轮（覆盖增强 + `-race`）」与 `CHANGELOG.md` `[Unreleased]` 首条。
- **WASM 编译门禁**已加进 `verify.sh`（实测通过）。
- **性能优化按 pprof 实测结论主动收窄**：CSS 选择器索引、解析缓存、离屏位图池化
  经数据判定为无收益或与基线无关，**明确不做**，理由写在 `BENCHMARK.md`。
- 仍未做（需要外部条件或点名）：fnOS FPK 集成、Lucide / circle-flags 真实语料对账、
  CI runner 安装、`cmd/verify` 与 `examples/*` 覆盖率、`internal/rasterx` 覆盖率
  （**当时快照**；`internal/rasterx` 覆盖率已于 2026-11-15 本轮补测落地 34.8% → 64.3%，
  见文末「rasterx 几何 / 形状补测」小节）。

## 本轮（分支 `feat/limits-and-coverage`）

- **CSS 规模上限**补齐（此前 `SECURITY.md` 自己标注「仍未限制：CSS 规则条数与单条规则长度」）。
  选择「降级 + 诊断」而非硬失败，与输入/尺寸上限的区别写进了 `GOSVG_TODO.md`。
- **并发渲染用例**补齐（此前无并发测试，`Diagnostics` 的锁只有代码审查保证）。
- **`cmd/verify` 覆盖率 0% → 62.3%**（重构为可测函数）、**`internal/rasterx` 0% → 3.9%**
  （补 `Matrix2D` 基础性质测试，fork 代码此前零覆盖）；根包 85.7% → 85.8%。
- 仍未做：`internal/rasterx` 的 `Gradient` / `Filler` / `Dasher` 测试、`examples/*` 覆盖、
  fnOS FPK 集成、真实语料对账、CI runner 安装（下载源已确认可达，等授权）。

## 本轮（分支 `feat/realworld-compat`）

按「全面能力增强分析」的第 1 批 + 第 2 批工具部分推进，全部由真实语料实测驱动：

| 缺口 | 修复 | 实测效果 |
|---|---|---|
| 非 UTF-8 encoding 声明硬失败 | `CharsetReader`（ASCII 透传 + Latin-1 展开，零新依赖） | Parse 失败 67 → 0 |
| 无效颜色值丢失继承 | `mergeStyle` + `isValidPaint` | `folder-off.svg` 等从全透明恢复 |
| `#gggggg` 被当黑色 | `parseColor` 加 `isHexDigits` 校验 | 无效声明改为忽略 |
| `<image>` data URI 需手动注入 | 内置 data URI 解码（不碰文件/网络） | 语料 B/C 的 7 次 skipped 消除 |
| 无真实语料回归手段 | `cmd/corpus` + `CORPUS_BASELINE.md` | 958 个 SVG：90.4% → 99.5% |

剩余 3 个全透明样例已逐个定位，均为语料本身的空 SVG（`empty.svg` 与
`annotation-noicon.svg` ×2）。原先的第 4 个 `lock-t_gZZsqx.svg`
（Figma 的 `<pattern>` + `<use>` + base64 `<image>`）已随 `<pattern>` 支持落地正常渲染，
详见 `CORPUS_BASELINE.md`。

**仍未做**：`<pattern>`、`strokeCapsules` 复用 Filler（pprof 靶点，占分配 38.9%）、
文本能力（letter-spacing / textLength / 换行 / shaping）、cairosvg 批量对账、
CI runner 安装（下载源已确认可达，等授权）。

## 本轮（分支 `feat/pattern-and-stroke-reuse`）

- **`<pattern>` 已实现**：真实语料 A 批 216/217 → **217/217**，958 个真实 SVG 整体
  954 → **955（99.7%）**，剩余 3 个全透明经核查均为语料本身的空 SVG，
  即**可渲染的真实 SVG 失败数为 0**。
- **`strokeCapsules` 复用 Filler 已实现**并做 A/B 实测：多 path 场景 B/op -58%、ns/op -19%；
  单 path 无变化。据此修正上一轮判断——「离屏位图池化」仍未做，但光栅器缓冲复用已做且有效。
- 实现中修掉两个自己引入的问题：`draw.DrawMask` 会用 `src.Bounds()` 裁剪目标矩形，
  导致「平铺 image + 一次 DrawMask」只画出左上角一块（64/256），改为按 tile 位置逐块绘制；
  以及 `tileW/tileH` 重复声明。
- 流程教训：从验证目录 `tar` 回传会覆盖本地未提交的文档改动（本轮踩到一次，已重做）。

**仍未做**：文本能力（letter-spacing / textLength / 换行 / shaping）、`filter` 与
`foreignObject`（Figma 毛玻璃，主体图形不受影响）、cairosvg 批量对账、
CI runner 安装（下载源已确认可达，等授权）、`cmd/render` 增强、fnOS FPK 集成。

## 本轮（分支 `feat/clip-rule`）

- **`clip-rule` 已实现**，推翻了此前 H-A 表里「不实现」的决策（当时估计「需把 fill-rule
  语义接进 clip 光栅路径」，实际只需映射到既有 even-odd 路径，几行代码）。
- **但实测真实影响为 0**：91 个文件含 clip-rule → 87 个 clipPath 块 → 10 个可能受影响 →
  对 5 个候选做修复前后逐像素对比，**差异全为 0**（它们的 clipPath 服务于被跳过的
  Figma `foreignObject`，或子路径不重叠）。属语义补全，golden 与语料成功率均未变化。
- **方法论教训（比这个特性本身更重要）**：用「属性出现频率」排优先级不可靠。
  本轮同时测得：`letter-spacing` 仅 2 次、`textLength`/`lengthAdjust` 0 次、
  `visibility` 0 个文件、`display` 只有无害的 `display:inline` —— 说明此前几轮按频率
  推断的优先级需要重新校准。正确的排序依据是**修复前后像素对比**或**与 cairosvg 对账**。
- 下一步建议改为：先做 cairosvg 批量对账（本机已有 cairosvg 2.9.1），用 IoU 差异
  定位「渲染出来了但不对」的文件，再据此决定实现哪个特性。

**仍未做**：cairosvg 批量对账、文本能力（实测低频，优先级下调）、`filter`（完整滤镜链
feBlend/feColorMatrix/feGaussianBlur/feFlood/feMorphology，成本与定位不符）、
CI runner 安装（等授权）、`cmd/render` 增强、fnOS FPK 集成。

## 本轮（分支 `fix/clip-transform-order-and-encode`）

- **cairosvg 批量对账落地**（117 抽样，中位 IoU 0.9982），并据此定位到一个
  **成功率指标完全看不到的真 bug**：`clip-path` / `mask` 与元素自身 `transform` 的
  求值顺序反了。修复后单文件 IoU 0.206 → 1.0000、A 批平均 0.9653 → 0.9947、**零退化**。
  这验证了上一轮的方法论修正：**对账比属性频率统计更能定位真实缺口**。
- **输出封装**：`RenderToWriter` / `EncodeImage`（PNG/JPEG）+ `RenderOptions.JPEGQuality`/`Ctx`，
  调用方不必再自己 `png.Encode`，也为 CLI 与 FPK 集成铺路。
- D 批（数位板图标）IoU 0.048–0.19 经逐项核查为**亚像素细线 AA + 字体差异**
  （`stroke-width:.25` 缩放后约 0.16px、重叠区 RGB 差仅 3–7），属已定案边界，不是 bug。

**新增待查**：B 批 `frame-icon.svg`（Inkscape 导出，IoU **0.0594**，gosvg 1464 vs
cairosvg 24562 不透明像素）——仅 5 个元素、1 条 `fill-rule="evenodd"` path +
`matrix()` transform，疑似真缺口（evenodd 与复杂 path 数据、或 `H`/`V`/`c`/`v`
相对指令与紧凑小数组合），下一步优先查它。

**仍未做**：`cmd/render` 增强（格式/宽高/stdin-stdout/批量/--strict）、WASM demo、
fnOS FPK 集成、CI runner 安装（需授权）、Gitea SQLite 锁（需授权）。

## 本轮（分支 `fix/no-viewbox-aspect-ratio`）

- **修复无 viewBox 非等比拉伸**：全量对账 848 样本实证平均 IoU 0.7870 → 0.9486、
  IoU<0.5 210 → 17、195 个文件提升、零退化。同时撤回上一版把 D 批判为
  「亚像素 AA + 字体已定案边界」的误判（依据的 RGB 差指标在位置错开时失真）。
- **even-odd + transform 假设被全量数据证伪**（高危 70 个中位 IoU 1.0000、仅 3 个 <0.5），
  撤回改 `internal/vector` 核心的计划。
- **SQLite 锁（零中断修法已生效）**：取消 33 个僵尸 queued run，38 个 run 全部
  `completed/cancelled`；日志里 `database is locked` 与 Actions cron 扫描显著减少。
  **根治仍需** `[database] SQLITE_BUSY_TIMEOUT` + 重启 Gitea（会中断服务，未获授权，未做）。

## 本轮（分支 `fix/viewport-physical-units`）

- **视口物理单位换算已落地并验证**：新增 `viewport.go` / `viewport_test.go`，
  `parser.go` 的 `d.width`/`d.height` 改用 `parseViewportLength`（96dpi，支持
  px/mm/cm/in/pt/pc/q）；元素几何保持 1:1 近似不变。`verify.sh` 在 .15 / Go 1.27.3
  全套 `ALL GREEN`（含 race / arm64 / wasm / 两个 fuzz 目标）。
- **上一版「D 批 54 个低分 = 物理单位未换算」的假设被全量数据证伪**：换算后
  平均 IoU 仅 0.9486 → 0.9507、D 批 54 → 52、退化 0，只救回两个 bamboo 文件。
  **不要再按「54 个待修」排期**。
- **剩余 52 个定性**：50 个抗锯齿 / 字体栅格化边界（bbox 与像素数都近似，
  IoU 0.70–0.77，非几何错误）→ 归入已定案边界；2 个填充差异
  （`toolbarButton-editorHighlight` 0.40、`altText_spinner` 0.45）→ 低优先级待查，
  不阻塞发版。

> ⚠️ **后续实测推翻（见 `CAIROSVG_IOU.md` 修正 6）**：「50 个 AA/字体边界 + 2 个填充差异」
> 的定性不成立；even-odd 缺陷实测影响 15 个低分中的 **11 个**，另有 19 个中低分文件受益，
> 共 **30 个文件**。用「bbox 与像素数都近似」判 AA 会漏掉内部几何错误。
- **文档一致性**：README / GOSVG_TODO / 包注释 / style.go 四处「所有单位 1:1 近似」
  已补上「根 `<svg>` 视口尺寸例外」的说明，避免文档与代码矛盾。

## 本轮（分支 `fix/viewport-relative-units-and-docs`）

盘点「还有哪些需要补齐」后的分类处置：

- **直接修（零决策）**：`parseViewportLength` 相对单位回归（v0.2.2 新引入，`100em` → 0 尺寸）；
  7 处文档自相矛盾 / 过时（诊断通道、覆盖率、Release 状态、`ResetFonts()`、分支保护、
  H-A 视口例外、修正 1 旧结论标注废弃）。全套 `verify.sh` 复验，`golden.sha256` 未变。
- **需用户点头（会改渲染字节 + 全量 IoU 表，未动）**：2 个经 PNG 复核确认为**整块缺失**的真 bug
  —— `altText_spinner`（`@media` 嵌套块内 `#hourglass{display:none}` 被当顶层规则应用）、
  `toolbarButton-editorHighlight`（even-odd 多子路径被挖空）。详见 `CAIROSVG_IOU.md` 修正 3。
  二者不属于「50 个 AA/字体已定案边界」，值得修，但要先交影响账。

> ⚠️ **后续实测推翻（见 `CAIROSVG_IOU.md` 修正 6）**：不是 2 个，是 11 个低分文件 + 19 个
> 中低分文件；「二者不属于 50 个 AA 边界」方向对，但计数与优先级判断已作废。
>
> ✅ **已由修正 7 落地**（2026-11-10 用户授权）：等价转换在解析层实现、光栅器核心零改动，
> 全量 848 样本 **30 提升 / 0 退化**、平均 IoU **0.949874 → 0.961272**、
> `IoU<0.5` 15 → 4。`altText_spinner`（`@media`）不在本次修复范围内，仍是待查项。
>
> ⚠️ **后续已修（修正 8 / PR #18，v0.2.3）**：`css.go` at-rule 深度扫描后该文件
> IoU 0.4549 → 0.9990；「仍是待查项」是修正 7 当时的快照，现已失效。
- **档 2 H-A 清单**：七项里 `clip-rule`、`vector-effect="non-scaling-stroke"`、
  `marker orient="auto-start-reverse"`、`<image>` 的 `preserveAspectRatio` 已推翻原决策、
  改为实现（clip-rule 实测 91 个文件 / 128 次使用但真实影响为 0；后三项均已加像素级
  回归测试，image PAR 另经 850 语料 A/B 验证 9 个 image 文件逐字节一致），
  其余 **三项**仍按原决策保持子集边界（用户需单独点名才做）；「元素几何单位
  真换算」与已落地的「视口尺寸换算」已在表中区分清楚。
- **档 3 外部条件项**：推送外部 Git（用户约束未解除）、Gitea SQLite 根治需重启服务（会中断，
  未获授权）、arm64 真机 runner、act_runner 注册（无二进制 / 出站不通）、CI 内 cairosvg 对账。
  这些不是代码缺口，**不因「继续修」而自动推进**。
- **`OPEN_SOURCE_CHECKLIST.md` 里本地可补的部分**：全历史敏感串扫描、许可证核对、
  私有地址清单、outside view 验证，仍待用户决定是否要在推送前逐项做；本轮未擅自对外动作。

## 本轮（依赖 / 盲点续查后的本地收口，2026-11-15）

按「继续分析仓库还有哪些依赖、哪些盲点没检查到」的盘点结果，取优先级最高的
两件 **P0 且完全本地可做** 的落地；凡涉及 `.15` 的写操作一律只出方案、不动手。

- **P0 · v0.3.0 主功能补回归保护**：此前 `strokeTextSpan` / `appendGlyphSegments`
  （PR #29）在仓库内零单测、7 个 golden 无一含 `text`+`stroke`，唯一验证是仓库外
  的 117 文件 IoU 子集。新增 `text_stroke_test.go`（6 个 Test）与
  `testdata/text_stroke.svg`（第 8 个金样）；`golden.sha256` 只多一行，其余 7 个
  哈希逐字未变（证明零渲染改动）。锚点 `test=232→238` / `golden=7→8` 同步。
- **P0 · CI 现状如实化**：查 Gitea API 得 **113 run / 0 success / 106 cancelled /
  7 queued（#107–#113，均为 v0.3.0 发布相关 push/PR）/ runners=0**。
  ⚠️ 以上为该轮取证快照，按纪律保留原文；事后 7 个 queued 已取消
  （113 全 cancelled / queued=0），当前状态见本文件末轮「CI 止血 + 发布清单」。
  `CONTRIBUTING.md` 原本已有「runner 数为 0、未在服务端真正执行」一句，本轮不改其
  结论、只补量化证据；`README.md` 开发段新增 IMPORTANT 块，明确「双 CI 配置同源」
  ≠「自动验证已生效」，ALL GREEN / IoU 数字均来自本地手动 `verify.sh`。
  CHANGELOG `[Unreleased]` 与 `GOSVG_TODO.md` 同步。
- **按代码实测更正两处文档**：①全仓无 `paint-order` 解析器，`text.go` 无条件
  「先描边、后填充」。此前 `GOSVG_TODO.md` / `CHANGELOG` 把 `paint-order: stroke`
  列为「已支持」属表述不准，已改；未改渲染行为。②（复核证伪旧结论）`text.go:30`
  `parseLengthOr(font-size, 12)` **不会**把 `font-size="0"` 当缺失回退默认 12：
  `parseLength("0")=(0,true)`→返回 0→`drawTextSpan` 的 `fontSize<=0` 跳过绘制，
  实测零字号渲染 0 像素（正确）。此前记为「零字号仍绘制」是对 `parseLengthOr`
  的错误推断，已更正，无需改代码。
- **既有语义（未改，非 bug）**：`font-size=48 / stroke-width=2` 时描边轮廓（288px）
  比字形填充（148px）更大（两条光栅路径不同，不可凭直觉断言「空心比实心小」）。
- **本轮明确未做（档 3 外部条件，需用户授权 / 点名）**：注册 act_runner（动 `.15`
  服务、GitHub 出站不通）、清理 7 个 queued run（写 `.15` Gitea）、Gitea SQLite
  根治（需重启服务，会中断）、arm64 真机测试、CI 内 cairosvg 对账、850 语料入仓、
  `internal/rasterx` 的 `Filler`/`Gradient`/`Dasher` 补测（当前覆盖 3.9%）、
  fork 上游 commit pin、汇编生成器找回 / 汇编冻结声明、NOTICE / dependabot。
  这些不因「按最优建议来」自动推进——本轮只交付本地可验证的两件 P0。
  ⚠️ **状态更新（后续轮次）**：这是该轮当时快照，按纪律保留原文；此后
  `internal/rasterx` 补测已落地（3.9% → 34.8%）、NOTICE 已新增、上游 pin 已补、
  7 个 queued run 已逐个取消。「注册 act_runner」「850 语料入仓」「arm64 真机」
  等至今仍未做。
- **本轮验证记录（诚实口径，勿写成 ALL GREEN）**：用 goproxy.cn 下载的
  Go 1.27.3 临时工具链（仅 `/tmp`、未装系统包、未动 `.15`）逐项实跑：
  `go mod verify` / `tidy -diff` / `gofmt -l` / `go vet` / `go test ./...`
  （根包 cover **87.1%**，原 87.0%）/ arm64 + wasm 交叉编译 / bench smoke /
  6 个 fuzz 目标（FuzzParse 30s、FuzzRender 20s、其余各 5s）/ `check_docs.sh
  **DOCS CHECK PASSED`** 全部通过。**唯一未跑项是 `go test -race`**：本机
  （临时 Go 工具链所在容器）无任何 C 编译器，`-race` 强制要求 cgo，报
  `-race requires cgo` / `C compiler "gcc" not found`。这是执行环境限制，
  **不是仓库缺陷、也不是本次改动引入**；要补跑需在有 gcc 的机器（如 `.15`
  或有 build-essential 的开发机）上执行 `sh ./scripts/verify.sh` 全套。
  仓库多处「零 cgo」指运行时零 cgo，未说明 `-race` 自身的 cgo 前提，
  属已知小盲点，本轮不扩范围。

## 本轮（P1/P2 本地收口，2026-11-15，接上轮）

- **rasterx 补测**：新增 15 个 Test，覆盖率 **3.9% → 34.8%**（档 1 的
  「Filler/Gradient/Dasher 测试」待办由本轮部分落地；仍非全覆盖）。
  查实两个构造陷阱（Points 全零 → t=NaN、Matrix 零值 → gradT 退化），
  根包构造已传 Matrix，运行期无恙。
- **`-race` 卡死整条门禁**（上轮遗留）已修：`verify.sh` 现在无 gcc 时
  SKIP race 并继续跑完 arm64/wasm/6 fuzz/docs，结尾标注 SKIPPED，
  **有 gcc 时行为不变**。本机试过 zig（下载超时 9/47MB、无 xz）、tcc
  （能编 .o 但缺 crt1.o/-lc）均不装系统包即不可用，故 race 仍未实跑；
  **这是执行环境限制，不是放宽门禁**。最终 `verify.sh` 退出码 0、FAIL=0。
- **上游 pin**：`internal/vector` 用本机 module cache 与 `x/image@v0.47.0/vector`
  逐文件硬比对，补上可核对锚点与差异表。⚠️ **推翻上轮「gen.go 缺失→汇编
  不可再生」**：上游同版本带 gen.go + 模板，且 acc_amd64.s 逐字节一致；
  fork 只是未随附、当前无需再生。`internal/rasterx` 上游无 tag，如实记
  最新 commit 2ab79fc（2022-07-30）、**不伪造引入点 hash**；与 2ab79fc
  的逐文件比对尚未做，仍属待办。
- **一致性/合规**：新增 NOTICE；去掉 fork 的 canonical import comment；
  更正 `x/sys` 为直接 require；`check_docs.sh` 加双 workflow 一致 +
  Go 版本单一来源两条机械门禁（1.27.3）；锚点 test=249。
- **仍未做（不因「继续按建议」自动推进）**：带 gcc 机器补跑 `-race`、
  rasterx 与 2ab79fc 逐文件比对、`internal/vector` 24.3% 与 cmd/* 低覆盖补测
  （⚠️ 两项事后均已完成，见本文件末节「cmd/* 补测」）、
  850 语料入仓、arm64 真机、CI runner 注册 / 7 queued run 清理（**写 `.15`，
  须用户明确授权**）、Gitea SQLite 根治（需重启，会中断）、提交/推 PR
  （对外动作，须用户点头）。本轮所有改动均为**本地、零渲染代码变更**。
  ⚠️ **状态更新（PR #32 合并后）**：这是该轮当时的快照；7 个 queued run 事后已
  逐个取消（非删历史，113 全 cancelled / queued=0）、提交与 PR 已完成并合并。
  「带 gcc 机器补跑 `-race`」「CI runner 注册」等**至今仍未做**，继续按档 3 条件。
  ⚠️ **再更新（覆盖增强轮，2026-11-15）**：`-race` 已在**本机**用一次性 `/tmp` gcc
  工具链跑通（`ALL GREEN` 无 SKIPPED，见下节），故「带 gcc 机器补跑」若指
  **稳定环境复现**仍未做、若指「`-race` 从未实跑」则已不成立；`internal/vector
  24.3%` 补测已完成（→83.4%）。⚠️ **cmd/* 补测（同轮后续，2026-11-15）**：4 个
  `main()` 已拆出可测 `cliMain(args) int`，新增 `internal/cmdtestutil`（仅测试用）
  与 4 个 `cli_test.go`（7 个顶层 Test + 子用例），覆盖率 62.3/42.2/37.2/16.7% →
  **81.8/85.3/85.3/88.5%**，`cliMain` 最高 100%、`compareBaseline` 100%、
  `printText` 90.9%；4 个 `main()` 仅剩 `os.Exit(...)` 一行为不可测边界。
  **CI runner 注册与 7 queued run 清理（写 `.15`）至今未做**，档 3 条件不变。

## 本轮（CI 止血 + 发布清单，2026-11-15，接上轮「最优方案」第 1+2 步）

- **第 1 步 · 止血**：两份 workflow 的 `on:` 从
  `push`（全分支 + `v*` tags）+ `pull_request` + `workflow_dispatch`
  **收窄为仅 `workflow_dispatch`**。此前 #107–#113 这 7 个 queued 正是
  自动触发在 runner=0 下堆出来的（该轮取证原文；事后已逐个取消，见下方
  「状态更新（PR #32 合并后）」）；收窄后不再新增僵尸 run。文件头注释写清
  「恢复自动触发」的两个前提（runners>0 且 GitHub action 可拉取）与步骤，
  `check_docs.sh` 的逐字一致门禁保证 `.gitea` / `.github` 两边同步。
  ⚠️ 注意：原先**已有** `workflow_dispatch`，本轮是「去掉 push/PR」而非新增。
- **第 2 步 · 发布清单正式化**：`CONTRIBUTING.md` 的「提交前必跑」补上
  「`-race` 需 gcc」的说明，并把**发版 / 合并 main 前必须在带 gcc 机器上
  跑到不带 SKIPPED 的 `ALL GREEN`、并在发布说明/PR 记执行机器与结果**
  写成硬要求（此前只散落在文档角落）。`README` IMPORTANT 块同步。
- **取证不变**：113 run / 0 success / 7 queued / runners=0、Gitea 28.3.0、
  runner 注册 token 端点 GET 404、GitHub 出站不通。
  ⚠️ 该组数字为当时取证快照；事后 queued 已清零（113 全 cancelled），
  runners=0 / 出站不通 / GET 404 三项结论至今有效。
- **明确未做**：7 个已存在的 queued run **未清理**（写 `.15`，须单独授权）；
  runner 未注册；`cb87d52` 与本轮 commit **未 push / 未开 PR**。
  ⚠️ **状态更新（PR #32 合并后）**：以上为该轮取证快照，按纪律保留原文；
  7 个 queued run 事后已逐个取消（113 全 cancelled、queued=0），两个 commit
  已推并经 PR #32 合并进 main。runner 仍未注册（此项至今有效）。
- **结论口径**：这不叫「修好 CI」，叫「停止让 CI 假装会跑」。真正恢复自动
  CI 仍是长期项，触发条件不变（见档 3）。

## 本轮（`internal/vector` 覆盖增强 + `-race` 首次真跑，2026-11-15）

- **覆盖增强（零渲染改动）**：新增 `internal/vector/rasterizer_extra_test.go`
  （12 个 Test），`internal/vector` 覆盖率 **24.3% → 83.4%**，`Draw`
  **33.3% → 100%**。补测对象不是「刷数字」而是两类真实裸奔语义：
  (1) fork 把 `floatingPointMathThreshold` 从 512 改成 0 后，
  `fixedLineTo`/`fixedAccumulate*` 成为正常调用不可达的死路径；
  (2) amd64 `haveAccumulateSIMD=true` 让标量 accumulate 全被 SSE4.1 汇编绕过。
  两类均用**同包白盒**直接锁定数值语义，另加 `TestDrawEvenOddSkipsUniformFastPath`
  回归锁（防 SIMD fast path 绕过 `accumulateMask` 导致镂空失效）。断言用
  「0 / 非零 / 相对关系 / 饱和 / fixed-floating 一致性」，避开 AA 相位误报。
  锚点 `test=249→261`、`GOSVG_TODO.md` 覆盖率段同步。
- **`-race` 硬要求首次真正满足**：本机（`Linux iStoreOS 6.12.112 x86_64` /
  `go1.27.3`）原本 `CGO_ENABLED=0`、无 gcc/clang；用 `apt-get download` 把
  gcc 14.2.0 + 运行库 + 头文件**仅解包到 `/tmp/gcc_probe`**（未 `dpkg install`、
  未改系统目录），wrapper 拼出工具链后 `CC` 指向它，`sh ./scripts/verify.sh`
  **rc=0 / `ALL GREEN`（无 SKIPPED）/ `DOCS CHECK PASSED`**；`-race` 全 7 包通过、
  零 data race，6 fuzz / arm64 / wasm / bench / fmt / vet / tidy-diff 全过。
  日志 `/tmp/verify_full2_1115.log`。执行机器与结果已登记进 `CHANGELOG.md`。
- **⚠️ 不改口的两条**：(1) 该 gcc 是一次性 `/tmp` 工具链、非稳定「带 gcc 机器」，
  `verify.sh` 的「无编译器则 SKIP」兜底与 `CONTRIBUTING.md` 硬要求照旧有效；
  在稳定环境复现仍属待办。(2) **CI 自动门禁仍未恢复**：runners=0、
  113 run 全 cancelled、GitHub 出站不通，「没有一次 PR/push 被服务端自动验证」
  至今有效，这次仍是**本地手动全绿**。档 3 触发条件不变。
- **明确未做（不因本轮自动推进）**：`cmd/*` 低覆盖补测（⚠️ 事后已在同轮后续
  完成，见下节）、850 语料入仓、rasterx 与 `2ab79fc` 逐文件比对、arm64 真机、
  CI runner 注册 / 7 queued run 清理（**写 `.15`，须用户明确授权**）、Gitea SQLite
  根治（需重启，会中断）、提交 / 推 PR（对外动作，须用户点头）。本轮所有改动均为
  **本地、零渲染代码变更**。

## 本轮后续（`cmd/*` 低覆盖补测，2026-11-15）

- **CLI 编排可测化（零渲染改动）**：4 个 `cmd/*/main.go` 的 `main()` 从「内联
  解析 + `os.Exit`」拆为 `os.Exit(cliMain(os.Args))` + `cliMain(args []string) int`，
  `cmd/corpus` / `cmd/iou` 改用局部 `FlagSet`（避免全局 `flag.CommandLine` 在多次
  测试间残留）。`cliMain` 逻辑与重构前逐分支等价，仅参数来源与退出方式不同。
- **补测内容**：新增 `internal/cmdtestutil`（仅 `_test.go` 引用的临时目录 / stdout
  捕获 helper，不进生产二进制）与 4 个 `cli_test.go`（**7 个顶层 Test** + 子用例）：覆盖
  参数不足 / 未知 flag / `-h` / 读写渲染失败 / 空目录 / 目录不可读 / JSON 落盘 /
  全参数组合、`printText` 全条件分支、`compareBaseline` 的
  improved / regressed / unchanged / ref 缺失 / base 缺失五条分支。
- **实测覆盖率**（`go test -count=1 -cover ./...`）：`cmd/verify` 62.3 → **81.8%**
  （`cliMain` 100%）、`cmd/corpus` 42.2 → **85.3%**（`printText` 90.9%）、
  `cmd/iou` 37.2 → **85.3%**（`compareBaseline` 100%）、`cmd/render` 16.7 →
  **88.5%**。**唯一 0% 是 4 个 `main()`**：只剩 `os.Exit(cliMain(os.Args))` 一行，
  是设计上的不可测边界，不用子进程刷数字。锚点 `test=261→268`。
- **明确未做（不因本轮自动推进）**：850 语料入仓、rasterx 与 `2ab79fc` 逐文件
  比对、arm64 真机、CI runner 注册 / 7 queued run 清理（**写 `.15`，须用户明确
  授权**）、Gitea SQLite 根治（需重启，会中断）、提交 / 推 PR（对外动作，须用户
  点头）。本轮所有改动均为**本地、零渲染代码变更**。

## 本轮（无效 paint 声明补诊断，2026-11-15，接 `feat/render-cli-enhance`）

承接上一轮 CLI 增强里的 `-strict`：若诊断通道本身漏报跳过事件，`-strict` 就是假安全网。
只读盘点全部 19 处 `addDiagnostic` 调用点后，确认主路径（display/visibility 为有意行为、
定义节点、mask/clip/gradient/image/marker/pattern 的能力缺失点）均已覆盖，**唯一真漏洞**
在颜色声明这一层：

- **漏洞**：`mergeStyle` 用 `isValidPaint` 把无效 `fill`/`stroke` 声明按 CSS 语义忽略并
  回退继承值（渲染正确，见 `colorfallback_test.go`），但这一「静默改写」零诊断。实测
  `fill="notacolor"` / `fill="000000"`（缺 `#`）/ `fill="rgb(999,0,0)"` 中前两者悄悄改画
  黑色、`-strict` 无从得知。`url(#不存在)` 虽已有 `gradient reference not found` 诊断
  （措辞偏 gradient，属误导但非「零诊断」），`resolvePattern` 静默返回 nil 也只在同一
  路径上，均不属于本次要补的洞，未动。
- **修复（纯只读、零渲染/像素改动）**：`renderNodeBare` 的 `mergeStyle` 之后加
  `reportInvalidPaints(ctx, n)`，对 `n.attrs["fill"/"stroke"]` 的最终生效值校验
  `isValidPaint`，为假则 `addDiagnostic`。放在 `renderNodeBare` 而非 `renderShape`，是为
  覆盖容器 `<svg>/<g>` 上经 `ctx.style` 影响整棵子树的继承值；显式排除纯定义节点
  （`defs`/`mask`/`clipPath`/gradient/`marker`/`pattern`/`stop`/元数据）以免误报。
  CSS / 内联 `style` 最终都写回 `n.attrs`，一处覆盖全部来源。`renderShape` / `text.go`
  分派链、`golden.sha256` 一行未动。
- **验证**：新增 `invalid_paint_diag_test.go` 7 个顶层 Test（无效有诊断、合法零误报、
  容器、内联 style、定义节点豁免、去重、像素逐字节不变）。`verify.sh` **ALL GREEN 含真
  `-race`、无 SKIPPED**（`CC=/tmp/gcc_probe/pathbin/gcc`）；锚点 `test=274→281`。
- **明确未做（不因本轮自动推进）**：未改 `CORPUS_BASELINE.md` 里 `gradient reference
  not found` 历史快照措辞、未动 `resolvePattern`、未碰 850 语料 / rasterx 逐文件比对 /
  arm64 真机 / CI runner 注册（写 `.15`，须授权）/ Gitea SQLite 根治（需重启，会中断）/
  推远端 / 开 PR / 合并（对外动作，须用户点头）。本轮仅本地提交在特性分支
  `fix/invalid-paint-diagnostics`。

## 本轮（修正 PR #35 合并 commit message + 补档，2026-11-15）

> 注：紧邻上一段末尾「推远端 / 开 PR / 合并（对外动作，须用户点头）」是**当时**的真实
> 状态，本段记录其后经用户逐次点头完成的动作，旧文字按「历史快照就近护注」纪律保留不改。

- **合并收口**：`fix/invalid-paint-diagnostics@10ac052`（含 `f4e748a` CLI 增强 +
  `10ac052` 诊断补漏两个 commit）经一次性 askpass 推远端、开 **PR #35**、
  `Do=merge` 合入 main。合并态 `e29af8e`，父 `0a29b62` + `10ac052`，
  merge 态 tree 与特性分支逐 blob 一致；合并态 `verify.sh` ALL GREEN 含真 `-race`、
  `check_docs.sh` PASSED（tag 后 15 提交、`[Unreleased]` 195 行、`test=281`）。
- **瑕疵**：merge 请求误带 `merge_title_field:"title"` 与空 `merge_message_field`，
  Gitea 把 `title` 当字面量写进 merge commit message（`git cat-file -p` 确认）。
- **修正（重写 tip，非改代码）**：`git commit-tree` 重建 commit 对象 → `3938cb6`，
  message 按 PR #34 模板写全；**tree `ffcb930f`、两父 `0a29b62`+`10ac052`、
  author/committer/时间戳逐字节一致，仅 message 不同**。改 message 必然换 hash，
  故 `--force-with-lease=refs/heads/main:e29af8e21f…` 推 main（lease 生效）。
  推前 `PATCH /branch_protections/main` 仅放宽 `enable_push`/`enable_force_push`
  两个布尔（GET 复核其余字段未动），推后**同一条命令立即恢复**，逐字段核实与备份
  除 `updated_at` 外完全一致。`v0.3.0` tag（`baa0c7d`）未动，open PR = 0，
  askpass 已删、`.git/config` 无凭据残留。
- **验证**：新 tip 与 `git fetch` 后的最终态各跑一次 `verify.sh`，均
  **ALL GREEN 含真 `-race` 无 SKIPPED + DOCS PASSED**；本地 = 远端 = `3938cb6`，干净。
- **已知残留（明确未修）**：Gitea PR #35 数据库 `merge_commit_sha` 仍为 `e29af8e2`
  （API 该字段只读，空 PATCH 与带字段 PATCH 均核实零副作用、值不变）；修正需改
  `.15` Gitea 数据库 / 重启，按用户禁令未做。仅影响 PR 页面 "merged as" 链接指向旧
  悬空 commit，代码/分支/tag/门禁均不受影响。以后合并请求不带
  `merge_title_field`/`merge_message_field` 即可避免。
- **本条补档本身**：纯文档（CHANGELOG `[Unreleased]` + 本段），常规 commit
  fast-forward 推 main，不需 force、不动保护规则、零代码/渲染/golden 改动。

## 本轮（修正 url(#id) paint 引用诊断，2026-11-15，接 PR #36 补档）

> 上一段「明确未做」里列的「`resolvePattern` 静默返回、`url(#不存在)` 只报 gradient
> 措辞误导」是**当时**的判断，本段记录其后经实测探针纠正与修复的结果；旧文字按
> 「历史快照就近护注」纪律保留不改。

- **纠正判断**：措辞误导只是表象。实测 `fill="url(#p)"`（合法 `<pattern>`）→
  蓝色正常绘制，却因 `resolveGradient` 里 `d.grads[id]==nil` 无条件报
  `gradient reference not found: p`。即**合法 pattern 被误报**，`-strict` 会对
  完全合法的文档错误退出 3 —— 比措辞严重得多，且可能是 `CORPUS_BASELINE.md`
  里 125 次该事件的来源之一（语料中若有 pattern 引用即命中）。
  `url(#不存在)` / id 指向 clipPath 等情形虽只有一条 gradient 措辞诊断，但**并非
  零诊断**，不属于「-strict 漏报」类漏洞。
- **修复（纯只读、零渲染/像素改动）**：
  - `resolveGradient` 改为纯解析，`d.grads[id]==nil` 时不报诊断直接返回 nil
    （同样的 id 可能是 pattern，职责不该在这里判断）。
  - `renderShape` 的 fill / stroke 分派链各加末端 `else`，调新增的
    `reportUnresolvedPaintRef(doc, v)`：仅当三条路都未命中、且 `gradientRef(v)`
    解析出非空 id 时，报 `paint reference not resolved: <id>`。描边侧额外加
    `style.strokeWidth > 0` 前提，避免 `stroke-width=0`（明确不描边）误报。
  - `resolvePattern` 保持静默（组合判断在分派处统一做，不重复报）。
    绘制分支一行未动，`golden.sha256` 一行未动。
- **边界（实测探针逐一坐实）**：`none` / `transparent` / 空串 / `stroke-width=0`
  零诊断；有效 gradient、有效 pattern、纯色零诊断；`stroke="url(#pattern)"`
  仍报既有 `stroke with pattern reference is not supported`（子集边界不变）；
  缺失 / 错误类型 id 报中性诊断；诊断按文本去重。
- **验证**：新增 `paint_ref_diag_test.go` 8 个顶层 Test 锁定上述全部不变量与
  像素逐字节不变；`verify.sh` **ALL GREEN 含真 `-race` 无 SKIPPED**；
  锚点 `test=281→289`。
- **历史文档处置**：`CORPUS_BASELINE.md` 的 `125 × gradient reference not found`
  是当时 850 语料的观测快照，**保留原文不改**（本轮改的是其后的代码行为，不是
  重跑语料）；`cmd/corpus` 的 `normalizeDiag` 按 `: ` 前缀通用归一化，不写死
  措辞，无需改动。若日后重跑 850 语料，该行应随之更新并注明基准版本。
- **明确未做（不因本轮自动推进）**：未重跑 850 语料（语料与参照 PNG 不在仓内，
  入仓涉及体积+版权须用户拍板）、未动 `internal/rasterx` 覆盖率、未碰 `.15`、
  未推远端 / 未开 PR / 未合并（对外动作须用户点头）。本轮仅本地提交在特性分支
  `fix/paint-reference-diagnostics`。

## 本轮（rasterx 几何 / 形状补测，2026-11-15，接 PR #37 之后）

纯 `_test.go` 补测，**零实现改动 / 零 golden / 零像素改动**，把
`internal/rasterx` 覆盖率从 **34.8%（实测 35.2%）→ 64.3%**，清掉档 1 里
「`internal/rasterx` 覆盖率」这条长期待办（仍非 100%，剩余未覆盖集中在
`Stroker` 的复杂 join/curvature 内部循环，属 fork 上游逻辑，不改实现）。

- 新增 `internal/rasterx/geomx_extra_test.go` **12 个顶层 Test**：
  向量工具（`Invert` / `turnStarboard90` / `turnPort90` / `DotProd` 双线性 /
  `Length`）、`Path` 全套（构建 / `ToSVGPath` / `String` / `Clear` / `Stop` /
  `AddTo` 回放与 `Stop(false)` 调用序列）、`ToLength`、`ClosestPortside`、
  `RadCurvature` 共线为 0、`CircleCircleIntersection` 相交与相离 / 内含、
  `CalcIntersect`、`RayCircleIntersectionF` 与 fixed 包装层。
- 新增 `internal/rasterx/shapes_extra_test.go` **15 个顶层 Test**：
  `ToFixedP`、`AddCircle` / `AddEllipse` / `AddRect`（含旋转）/ `AddRoundRect`
  （退化与倒角）、椭圆参数化 `ellipsePrime` / `ellipsePointAt`、
  `FindEllipseCenter` 对称性、`AddArc` 末点、`MatrixAdder` 变换与 `Reset`、
  `NewStroker` 默认值、`GapToCap` 与 5 种 cap、`Dasher` 曲线方法不 panic。
- 断言一律用数学恒等式 / 结构性质 / 明确边界，不写死像素或浮点尾数。
  实测坐实两处「注释直觉 ≠ 实现」：`RayCircleIntersectionF` 的有效方向与
  函数注释的直觉相反（按真实行为断言、不改实现）；`FindEllipseCenter`
  在 r=5、半弦长=3 时圆心是 (0,±4) 而非原点（弦垂直平分线 + 半径约束）。
- 锚点 `test=289→316`；`verify.sh` ALL GREEN（含真 `-race` 无 SKIPPED）。
- **明确未做（不因本轮自动推进）**：850 语料入仓 / 重跑（语料与参照 PNG 不在
  仓内，入仓涉及体积 + 版权须用户拍板，本机亦无 `cairosvg`）、CI runner 安装
  （写 `.15` 须单独授权）、`examples/*` 覆盖、fnOS FPK 集成。
