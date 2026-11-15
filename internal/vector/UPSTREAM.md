# internal/vector — 上游与本地改动

- **上游**：[`golang.org/x/image/vector`](https://pkg.go.dev/golang.org/x/image/vector)
- **许可证**：BSD-3-Clause，© The Go Authors（见同目录 `LICENSE`）
- **引入**：2026-11-07，随 gosvg 初始骨架 fork 进 `internal/`
- **上游锚点（2026-11-15 补 pin）**：`golang.org/x/image@v0.47.0`
  （模块发布时间 2026-10-09T15:57:36Z，来自 goproxy `.info`）。
  本机 module cache `golang.org/x/image@v0.47.0/vector/` 可逐文件硬比对；
  本次比对结果见下「逐文件差异」。此前只写「2026-11-07 引入」而无版本 pin，
  属已知盲点，现补上**可核对的版本锚点**（未拿到上游 git commit hash，
  但 module 版本号 + module cache 内容已足以复现）。
- **用途**：AA 光栅器（`Rasterizer`）、fixed/floating 扫描、`acc_amd64.s` SSE4.1 SIMD 加速；
  非 amd64 平台走 `acc_other.go`。`haveAccumulateSIMD` 依赖 `golang.org/x/sys/cpu`
  （`acc_amd64.go` 直接 import 它，故 `go.mod` 里 `x/sys` 是**直接 require**，
  非 indirect；非 amd64 / appengine / noasm 走 `acc_other.go`，该分支不引用
  `x/sys`，但 `go.mod` 仍按全平台统一声明）。

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

## 逐文件差异（对 `x/image@v0.47.0/vector/`，2026-11-15 实测）

| 文件 | 状态 |
|---|---|
| `acc_amd64.go` / `acc_other.go` / `acc_amd64.s` | **逐字节一致**（差异行 0）；汇编与上游同版完全相同 |
| `vector.go` | `UseEvenOdd` 字段、`accumulateMask` / `Draw` 分支、`floatingPointMathThreshold = 512 → 0`（强制走 floating 路径）、去掉 canonical import comment（fork 进 `internal/` 后误导性强）|
| `raster_fixed.go` | 追加 `fixedAccumulateMaskEvenOdd`（27 行）|
| `raster_floating.go` | 追加 `floatingAccumulateMaskEvenOdd`（30 行）|
| `evenodd_test.go` | fork 新增，上游无 |
| 上游有、fork 未随附（测试与生成器）| acc_test.go、example_test.go、vector_test.go、gen.go、gen_acc_amd64.s.tmpl（均为上游目录内文件，非本仓路径，故不用反引号引用）|

⚠️ **更正上一轮盲点结论**：此前记「fork 内缺上游汇编生成器 → acc_amd64.s 不可再生」。
实测上游 x/image@v0.47.0/vector/ **同时带有 gen.go 与
gen_acc_amd64.s.tmpl（均为上游目录内文件，非本仓路径），且 acc_amd64.s 与上游逐字节一致**。
准确说法是：fork 未随附生成器，但汇编并非「不可再生」——可从同版本上游模块缓存
取回生成器复现；且当前无需再生（汇编零改动）。`//go:generate` 指令在本 fork 内
仍是死的，这是**可维护性提示**，不是资产丢失。`.s` 只有旧式 `// +build`、
`.go` 侧已是 `//go:build` 的不一致同样存在于上游同版，非本 fork 引入。

## 已知边界

libart 扫描线 AA 对「恰好穿过像素角的 45° 斜线」有 ±1px 覆盖相位歧义，
fill 与 stroke 同源受影响。经回退实验定案为已知边界，不修（详见 `GOSVG_TODO.md`）。

## 同步策略

按需手动合并上游修复；合并后必须重跑 `go test ./...`（含 golden 回归）与 45°/cap 类用例。
任何新改动都要追加到本文件。
