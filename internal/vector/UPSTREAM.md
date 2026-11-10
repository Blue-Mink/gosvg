# internal/vector — 上游与本地改动

- **上游**：[`golang.org/x/image/vector`](https://pkg.go.dev/golang.org/x/image/vector)
- **许可证**：BSD-3-Clause，© The Go Authors（见同目录 `LICENSE`）
- **引入**：2026-11-07，随 gosvg 初始骨架 fork 进 `internal/`
- **用途**：AA 光栅器（`Rasterizer`）、fixed/floating 扫描、`acc_amd64.s` SSE4.1 SIMD 加速；
  非 amd64 平台走 `acc_other.go`。`haveAccumulateSIMD` 依赖 `golang.org/x/sys/cpu`
  （这是 `go.mod` 里 `x/sys` 间接依赖的来源）。

## 本地改动

**even-odd winding 真实现**（上游 `x/image/vector` 仅支持 non-zero）：

1. `Rasterizer` 增加 `UseEvenOdd` 字段。
2. `accumulateMask` 按 even-odd 分支，禁用 SIMD，走新增的
   `floatingAccumulateMaskEvenOdd` / `fixedAccumulateMaskEvenOdd`；winding 数取
   `ceil(|acc|/unit)`，奇数填充、偶数透明。
3. `Draw` 在 even-odd 时跳过 `*image.Uniform` fast path（否则 SIMD rasterize 绕过
   `accumulateMask`）。
4. 归属更正：`ScannerGV.SetWinding` 属于 **`internal/rasterx/scan.go`**（另一个 fork），
   本目录 `internal/vector` 并不含该方法；相关改动见 `internal/rasterx/UPSTREAM.md`。
   上面 1–3 条（`Rasterizer` / `accumulateMask` / `Draw`）才是本目录的实际改动。

## 已知边界

libart 扫描线 AA 对「恰好穿过像素角的 45° 斜线」有 ±1px 覆盖相位歧义，
fill 与 stroke 同源受影响。经回退实验定案为已知边界，不修（详见 `GOSVG_TODO.md`）。

## 同步策略

按需手动合并上游修复；合并后必须重跑 `go test ./...`（含 golden 回归）与 45°/cap 类用例。
任何新改动都要追加到本文件。
