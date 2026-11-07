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

## 档 2 — 需用户拍板（本轮未动）

| 项 | 需要决定 |
|---|---|
| A5/E1 module 路径 | `go.mod` 保留 `github.com/Blue-Mink/gosvg`（当前选择）还是改为私有 Gitea vanity 路径？改路径是破坏性变更 |
| E1 开源收尾 | GitHub `Blue-Mink/gosvg` 是否已存在、`go get` 是否可用（本环境外网不可达，无法验证） |
| A3/G4 单位语义 | 是否真做 `pt/cm/mm/in/em/%` 换算？会改变渲染字节与既有 IoU |
| B1 Diagnostics 形态 | `Document.Diagnostics()` / `RenderResult.Warnings` / `--strict` 三选一 |
| F1/F2 API 扩展 | 是否新增 `RenderWithOptions(ctx, data, opts)`（保留旧签名）；渲染选项（背景色/超采样/尺寸上限） |
| G1/G2 资源上限 | 最大尺寸（如 8192²）、超限返回错误还是 clamp、mask/clip 嵌套层数上限 |
| H-A 子集内语义 | `clip-rule` / `vector-effect` / `<image>` 的 `preserveAspectRatio` / `auto-start-reverse` / CSS `inherit` / `use` 的 width/height 是否实现 |
| E3 版本发布 | 是否打首个 tag / release、版本号 |
| 分支策略 | 直推 main 还是走分支 + 合并 |

## 档 3 — 本地补不了 / 受约束，只能留档

- **E1**：用户约束「不推外部 Git」，GitHub 侧状态无法验证 → 文档标注为未来事项。
- **C4**：ARM64 **真机测试**需 arm64 runner 或 qemu，.15 只有 x86_64；当前只能保证交叉编译通过。
- **C1**：CI 内跑 cairosvg 对账需 Python + cairosvg；当前用「gosvg 自身像素哈希」做回归，
  IoU 表仍是本地一次性数据。
- **D4**：`go 1.27.3` 的低版本兼容性无低版本 Go 环境可验。
- **H-B**：`filter` / `pattern` / `foreignObject` / 动画等属**明确子集边界**，按项目定位不补，
  只需在 A2 声明（已完成）。
