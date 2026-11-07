# Changelog

所有 notable 变更记录在此。提交信息沿用 Conventional Commits
（`feat:` / `fix:` / `docs:` / `chore:` / `test:`）。

## [Unreleased] — 2026-11-07

首个可开源基线，同日 29 次提交。

### Added
- 纯 Go SVG 子集渲染器：`Render` / `RenderTo` / `Parse` / `Document` 主 API
- fill/stroke（胶囊体并集精确轮廓）、even-odd winding、gradient（fill + stroke）
- mask（luminance，units/region/transform）、clip-path、tspan、marker、`<image>`（回调注入）
- CSS `<style>` + class/id/类型选择器、hsl()/现代颜色、dasharray/dashoffset
- preserveAspectRatio 9 种 align + meet/slice/none、MeasureText/MeasureTextEx
- 可选系统字体目录 `AddFontDir`
- 5 个 fuzz 目标、97 个测试函数、Gitea + GitHub 双 CI 门禁
- `cmd/render` CLI、`cmd/verify` 对账工具、`testdata/` 样例

### Fixed
- path1 小半径 arc stroke 的 IoU 0.9443 → 0.9987（替换 rasterx Stroker）
- 45° 斜线 AA 相位歧义定案为已知边界（附回退实验依据）
- 审计修复 6 项渲染 bug、`use` 循环栈溢出、fuzz 抓到的两个死循环

### 工程补齐（本次）
- 重写 `GOSVG_TODO.md` 为当前状态单一事实来源；补全「明确不支持」清单
- 新增 CHANGELOG / CONTRIBUTING / SECURITY / examples / UPSTREAM 记录
- CI 加 `-race` / fuzz 冒烟 / arm64 交叉编译 / Go 缓存
- 新增 Example / Benchmark / golden 像素哈希回归基线（6 个确定性样例）/ cmd 测试
- `parseFloat` 拒绝 NaN/Inf、`parseLength` 单位白名单
- 核对 `internal/rasterx/stroke.go` 依赖链并写入 `UPSTREAM.md`（经编译验证不能删）

> 尚无正式 tag / release，版本号待定（见 `GOSVG_GAP_TODO.md` 档 2）。
