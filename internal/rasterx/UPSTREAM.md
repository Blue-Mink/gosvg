# internal/rasterx — 上游与本地改动

- **上游**：[srwiley/rasterx](https://github.com/srwiley/rasterx)
- **许可证**：MIT，© 2018 the rasterx Authors（见同目录 `LICENSE`）
- **引入**：2026-11-07，随 gosvg 初始骨架 fork 进 `internal/`
- **上游锚点（2026-11-15 补记，诚实口径）**：上游仓库**无任何 tag / release**，
  GitHub API 查到的最新 commit 为 `2ab79fcdd4ef8b552e5b4e4b43d24f74fc87aa61`
  （2022-07-30，即上游自 2022 起未再改动）。本仓**无法证明** 2026-11-07 引入时
  对应的是哪个具体 commit（当时未记录、上游也无版本可 pin），故此处只登记
  「上游已冻结 + 可核对的 HEAD」，**不伪造引入点 hash**。
- **逐文件比对（2026-11-16 已完成，替代上条「尚未做」待办）**：从 goproxy.cn 取上游
  伪版本 `v0.0.0-20220730225603-2ab79fcdd4ef`（即 `2ab79fc`，proxy `.info` 时间与
  commit 日期一致），与本 fork 8 个生产 `.go` 文件逐文件 `diff`。结果：**均可核对，
  改动集中在下面「本地改动」列出的几处 + 若干 `//X`→`// X` 注释格式（无语义）**，
  无其它隐藏分叉。逐文件差异行数（`diff | grep -c '^[<>]'`，含注释）：
  `dash.go` 2、`fill.go` 8、`geomx.go` 11、`gradient.go` 4、`matrix.go` 10、
  `scan.go` 5、`shapes.go` 2、`stroke.go` 2。比对素材在
  `/tmp/rasterx_upstream_cmp/`（临时目录，非仓库资产）。
- **用途**：fill 扫描器（`Filler` / `ScannerGV`）、`Gradient`、`Dasher`、`Matrix2D`、
  `ApplyOpacity` 等几何工具。**gosvg 的 stroke 不走本包的 Stroker 路径**（改用自研胶囊体并集，见根目录
  `stroke.go`）。但本包 `stroke.go` 定义的 `Stroker` / `CapFunc` / `GapFunc` /
  `JoinMode` 及 Dasher 所需字段被 `dash.go`、`shapes.go` 依赖，**不能删除**
  （2026-11-07 删除后编译失败实测确认）。

## 本地改动

1. `fill.go`：`CubeTo` / `QuadTo` 的 `lineF` 回调把 `fixed.Int26_6(ex)` 直接截断 float32，
   每个 flatten 顶点偏移 ±0.5 fixed，产生方向性噪声 → 改为 `+0.5` **四舍五入**。
2. `fill.go`：`CubeTo` / `QuadTo` 的 `const tol` 由上游 `3` 改为 `100`，tol（flatten 容差）
   语义由用户单位改为**像素**单位。
3. `scan.go`：import 由 `golang.org/x/image/vector` 换成本仓 `internal/vector`；
   `ScannerGV.SetWinding` 由上游 **no-op**（注释「scanner gv does not support even-odd
   winding」）改为 `s.r.UseEvenOdd = !useNonZeroWinding`，把 SVG 的 `fill-rule="evenodd"`
   接到 fork vector 光栅器的 even-odd winding 实现。这是 fork even-odd 能力的接线点之一。
4. `geomx.go`：`ToLength` 由上游的浮点版（`pX*lnF/pLen`）改写为整数版
   （`lp := Length(p); p.X*ln/lp`）。**多数输入数学等价，但非逐位一致**：当 `lp` 因
   固定点截断而偏小（极短向量，如 `p=(1,1)` fixed 单位、`ln=64`），fork 版得 `(64,64)`、
   上游浮点版得 `(45,45)`，短向量方向放大失真。属 fork 既有语义，**本轮不修**
   （改动会动渲染字节，需 golden + cairosvg 双兜底）；仅在此记录，供日后溯源。
5. `gradient.go` / `dash.go` 等按 gosvg 的坐标系与样式层做了接线适配。
   `dash.go` / `shapes.go` / `stroke.go` / `matrix.go` / `gradient.go` 除
   `//X`→`// X` 注释格式外与上游一致（逐文件比对确认）。

## 同步策略

按需手动合并上游修复；合并后必须重跑 `go test ./...`（含 `testdata/golden.sha256` 回归）
与 `GOSVG_TODO.md` 中的 cairosvg IoU 对账。任何新改动都要追加到本文件。
