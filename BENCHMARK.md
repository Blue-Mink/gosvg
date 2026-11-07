# 性能基线

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
