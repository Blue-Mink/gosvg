# 性能基线

> **版本状态**：本文是 2026-11-10 的性能基线快照（Go 1.27.3 / .15 NAS），表中
> ns/op、B/op、allocs/op 与文末结论均为**当时实测**，不代表最新版本读数；重新
> 测量用 `go test -run '^$' -bench . -benchmem -benchtime 2s .`。

## 实测环境

- 机器：.15 NAS，AMD Ryzen 7 5825U with Radeon Graphics
- Go 1.27.3 / linux / amd64
- 命令：`go test -run '^$' -bench . -benchmem -benchtime 2s .`
- 样例：512×512，Lucide 风格线条图标路径

## 结果

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| BenchmarkRender | 8,873,903 | 5,367,766 | 292,168 |
| BenchmarkParse | 3,475 | 2,656 | 41 |
| BenchmarkRenderDocument | 9,152,628 | 5,365,127 | 292,127 |

> 同一台机器多次跑会在 **8.9–11.2 ms/op** 之间波动（CPU 频率/调度），数字只作相对基线。

## 结论（修正了上一轮的判断）

- `Parse` 只占约 **3.5 µs**，相对约 9 ms 的渲染可忽略（< 0.05%）。
- `BenchmarkRenderDocument`（复用已解析的 `Document`）与 `BenchmarkRender`（每次重新解析）
  差距在噪声范围内 → **解析缓存 / `Document` 复用不是优化方向**，上一轮把「Parse 每次重建」
  列为性能缺口是误判。
- 真正的开销在**渲染/光栅化**：约 292k 次分配、5.37 MB/op。后续优化应聚焦
  离屏 mask/clip 位图池化、光栅器分配、`clearGhostRGB` 全图遍历、CSS 规则匹配索引。

## 本轮新增的两个 benchmark

- `BenchmarkParse`：只解析不渲染，用于量化 `Parse` 在 `Render()` 全程里的占比。
- `BenchmarkRenderDocument`：复用已解析的 `Document` 反复渲染，用于量化
  「解析 + 渲染」与「仅渲染」的差距，判断 `Document` 复用 / 解析缓存的价值。

合并后请在同机重跑并更新上表（数字与 CPU / Go 版本强相关，只作**同一台机器上的
相对基线**，不要跨机比较）。

## 注意事项

- `scripts/verify.sh` 用 `-benchtime=1x` 做 benchmark 冒烟，只保证能编译运行，
  **不设性能门槛**（CI runner 与本地机器差异大）。
- 若做性能优化（离屏位图池化、CSS 选择器索引、Parse 缓存、`clearGhostRGB` 优化），
  改动前后各跑一次，用 `benchstat` 对比后再合并。

## 内存分配热点（pprof `-alloc_space`）

命令：`go test -run '^$' -bench '^BenchmarkRender$' -benchtime 3s -memprofile mem.prof .`
样本：417 次迭代，累计分配 2642 MB（单次约 5.37 MB / 292k allocs）。

| flat 占比 | 函数 |
|---|---|
| 38.9% | `internal/vector.(*Rasterizer).setUseFloatingPointMath` |
| 19.8% | `image.(*NRGBA).At` |
| 19.7% | `internal/vector.(*Rasterizer).accumulateMask` |
| 19.4% | `image.NewNRGBA` |

调用链：`newRenderContext` 占 39.6%；`strokeCapsules → strokeColor → stroke → renderShape` 占 60.3%。

**结论：开销几乎全在渲染/光栅化，解析与样式匹配可忽略。** 据此明确以下判断：

- **不做 CSS 选择器索引**：`applyCSS` 只在 `Parse` 跑一次，而 `Parse` 仅 3.5 µs
  （< 单次渲染的 0.05%）。加索引是无收益的复杂度。
- **不做解析缓存 / 强制 Document 复用**：实测 8.87 vs 9.15 ms，差异在噪声内。
- **暂不做离屏位图池化**：基线样例不含 mask/clip，池化对当前数字无影响；
  要做应先为「大批量带 mask/clip 渲染」单独建 benchmark。
- 后续若继续优化，方向是 `Rasterizer` 缓冲区/实例复用（`bufF32`/`bufU32` 已按 cap 复用，
  但每次 `newRenderContext` 都新建 Rasterizer）与光栅器分配。这属于 fork 核心，
  必须靠 golden 回归 + cairosvg 对账兜底后再动。

## strokeCapsules 复用 Filler 的 A/B 实测

`stroke.go` 原先每次 `strokeCapsules` 都新建 `ScannerGV` + `Filler`（512px 下每次分配
约 1MB float32 缓冲）。改为 `renderContext.strokeFillerFor` 按目标图缓存复用后，
同机 A/B（`-benchtime 2s`）：

| Benchmark | 指标 | 复用 | 不复用 | 变化 |
|---|---|---|---|---|
| `BenchmarkRenderMultiStroke`（60 条 stroke path，512px） | B/op | **88.7 MB** | 212.5 MB | **-58%** |
| 同上 | ns/op | **570 ms** | 701 ms | **-19%** |
| 同上 | allocs/op | 21,118,737 | 21,119,093 | ≈0 |
| `BenchmarkRender`（单条 stroke path） | 全部指标 | 292,168 allocs / 5.37 MB | 完全相同 | **无变化** |

**读数要点**：收益体现在 **B/op（字节）而非 allocs/op（次数）**——filler 创建只是几次
分配，但每次约 1MB。单 path 场景一次渲染只调用一次 `strokeCapsules`，复用无从生效，
因此 `BenchmarkRender` 数字不变（也没有恶化）。

据此更新上一轮判断：「离屏位图池化」仍未做，但**光栅器缓冲复用已做且实测有效**；
pprof 里 `setUseFloatingPointMath` 占 38.9% 指的是分配**字节**占比，与该结论一致。
