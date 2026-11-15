# internal/rasterx — 上游与本地改动

- **上游**：[srwiley/rasterx](https://github.com/srwiley/rasterx)
- **许可证**：MIT，© 2018 the rasterx Authors（见同目录 `LICENSE`）
- **引入**：2026-11-07，随 gosvg 初始骨架 fork 进 `internal/`
- **上游锚点（2026-11-15 补记，诚实口径）**：上游仓库**无任何 tag / release**，
  GitHub API 查到的最新 commit 为 `2ab79fcdd4ef8b552e5b4e4b43d24f74fc87aa61`
  （2022-07-30，即上游自 2022 起未再改动）。本仓**无法证明** 2026-11-07 引入时
  对应的是哪个具体 commit（当时未记录、上游也无版本可 pin），故此处只登记
  「上游已冻结 + 可核对的 HEAD」，**不伪造引入点 hash**。若日后要严格溯源，
  需拿本 fork 文件与 `2ab79fc` 逐文件硬比对才能确认是否一致——该比对尚未做，
  属待办（对比 `internal/vector` 已完成的 `v0.47.0` 逐文件比对）。
- **用途**：fill 扫描器（`Filler` / `ScannerGV`）、`Gradient`、`Dasher`、`Matrix2D`、
  `ApplyOpacity` 等几何工具。**gosvg 的 stroke 不走本包的 Stroker 路径**（改用自研胶囊体并集，见根目录
  `stroke.go`）。但本包 `stroke.go` 定义的 `Stroker` / `CapFunc` / `GapFunc` /
  `JoinMode` 及 Dasher 所需字段被 `dash.go`、`shapes.go` 依赖，**不能删除**
  （2026-11-07 删除后编译失败实测确认）。

## 本地改动

1. `fill.go`：`CubeTo` / `QuadTo` 的 `lineF` 回调把 `fixed.Int26_6(ex)` 直接截断 float32，
   每个 flatten 顶点偏移 ±0.5 fixed，产生方向性噪声 → 改为**四舍五入**。
2. `fill.go`：tol（flatten 容差）语义由用户单位改为**像素**单位。
3. `gradient.go` / `dash.go` 等按 gosvg 的坐标系与样式层做了接线适配。

## 同步策略

按需手动合并上游修复；合并后必须重跑 `go test ./...`（含 `testdata/golden.sha256` 回归）
与 `GOSVG_TODO.md` 中的 cairosvg IoU 对账。任何新改动都要追加到本文件。
