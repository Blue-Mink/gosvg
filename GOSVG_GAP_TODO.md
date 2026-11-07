# GOSVG 缺口清单与补齐进度

> 由 2026-11-07 两轮代码审阅合并而成。状态：**档 1 已在本轮补齐**；档 2 需用户拍板，
> 档 3 需外部条件或受「不推外部 Git」约束。

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
- **`AddFontDir` 全局状态污染测试**：`font_test.go` 注册系统字体后，包级 `fontRegistry`
  影响同进程内后续所有 `Render`。实测 `s60s.svg`（带 `<text>`）单独跑 golden 与跑完整
  `go test ./...` 的像素哈希不同（`bb150194…` vs `dc09dbc6…`），其余 6 个纯图形样例一致。
  当前处置：golden 基线排除 `s60s`。彻底修复需给字体注册加隔离 / `Reset`（并入档 2 F4）。

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
| E3 版本发布 | **合并 PR 后再打 `v0.1.0`** | 不在分支上打 tag，避免 tag 指向未合并的提交 |
| 分支策略 | **已改为分支流** | 新建 `chore/close-gaps`，后续改动走分支 + PR，main 不再直推 |

### H-A 逐项决策（为什么不实现）

| 特性 | 决策 | 理由 |
|---|---|---|
| `clip-rule` | 不实现 | 需把 fill-rule 语义接进 clip 光栅路径，会动 clip 相关对账 |
| `vector-effect="non-scaling-stroke"` | 不实现 | stroke 宽度在 viewBox 变换里统一缩放，单独豁免要改 stroke 管线 |
| `<image>` 的 `preserveAspectRatio` | 不实现 | 需把 meet/slice 裁剪逻辑引入 image 缩放路径 |
| `marker orient="auto-start-reverse"` | 不实现 | 需在起点反转切线，牵动 `renderMarkers` |
| CSS `inherit` | 不实现 | CSS 级联层需引入「继承值」概念，超出简单选择器子集 |
| `use` 的 `width/height` | 不实现 | 需按 symbol/view 语义做视口裁剪，牵动 `renderUse` |
| 长度单位真换算 / `%` 基准 | 不实现 | 会改变现有全部含单位样例的渲染字节与 IoU 表 |

> 共同理由：这些都属于**扩大子集**，会改变渲染结果、要求重跑 cairosvg 对账与 golden
> 基线，已超出「零决策、不破坏语义」的补齐范围。需要哪个请单独点名，单独做。

## 档 3 — 处置结论与触发条件

| 项 | 当前处置 | 触发条件 / 具体做法 |
|---|---|---|
| E1 GitHub 侧确认与推送 | `OPEN_SOURCE_CHECKLIST.md` 留清单 | 用户解除「不推外部 Git」后：按清单核 → 建外部仓 → 加 remote → push main + tag |
| C4 ARM64 真机测试 | CI 已有 `GOARCH=arm64 go build ./...` 交叉编译门禁 | 有 arm64 runner 时加 job `runs-on: [self-hosted, linux, arm64]` + `go test ./...`；或用 `qemu-user-static` 跑交叉编译出的测试二进制。当前 .15 只有 x86_64，**不引入外部 action** 以保持 CI 稳定 |
| C1 CI 内 cairosvg 对账 | `testdata/golden.sha256` 做 gosvg 自身像素哈希回归 | 愿意加 CI 依赖时：`apt install python3-cairosvg` + 补一个 IoU 脚本（仓库暂无），把 512px 对账表变成门禁 |
| D4 低版本 Go 兼容 | **决策：锁定 Go 1.27，不追求低版本兼容** | `go.mod` 声明 `go 1.27.3`；若未来要降级，下调 go 指令并给 CI 加版本矩阵（`1.26.x` 等）验证 |
| H-B（filter/pattern/foreignObject/动画） | **明确子集边界，不补** | 已在 README / 包注释 / `GOSVG_TODO.md` 三处声明，无需动作 |

> 档 3 的共同点：不是「做不到」，而是**受用户约束或缺少外部条件**。每项都给了替代
> 方案与触发条件，条件满足时可直接按表推进，不需要重新讨论范围。

## 本轮（分支 `chore/close-gaps`）

- 档 2 的可落地项已实现：`RenderWithOptions` / `RenderOptions` / `Diagnostics` / 尺寸与嵌套上限。
- 默认路径渲染字节不变（旧 API 转调新入口、零值选项等价于原行为），
  `testdata/golden.sha256` 不重生成；由 `go test ./...` 验证。
- 所有改动走分支 + PR，main 不直接推。
- 已推到 main 的第一批 6 个 commit 保留（未改写已发布历史）；若要连第一批也撤回，
  需要 force push 回退 main，须用户另行点名授权。
