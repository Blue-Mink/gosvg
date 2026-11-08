# 真实语料基线

用 `cmd/corpus` 对 **.15 上 958 个真实世界 SVG** 批量渲染的结果，作为能力回归的对照基线。
与 `testdata/golden.sha256`（仓库内 6 个固定样例的像素哈希）互补：golden 防「无意改变渲染」，
corpus 回答「真实 SVG 有多少能正确渲染、还缺什么能力」。

## 基线（2026-11-08，Go 1.27.3 / linux-amd64，`-size 64`）

| 批次 | 语料来源 | 数量 | 渲染出内容 | 全透明 | Parse 失败 | Render 失败 |
|---|---|---|---|---|---|---|
| A | fnOS 面板图标 `/usr/trim/www` | 217 | 216（99.5%） | 1 | 0 | 0 |
| B | node_modules 图标库 | 355 | 354（99.7%） | 1 | 0 | 0 |
| C | 应用配置 SVG | 127 | 127（100%） | 0 | 0 | 0 |
| D | docker overlay 混合来源 | 259 | 257（99.2%） | 2 | 0 | 0 |
| **合计** | | **958** | **954（99.5%）** | **4** | **0** | **0** |

崩溃（panic / hang）数：**0**。

## 本轮修复带来的变化

| 指标 | 修复前 | 修复后 |
|---|---|---|
| 整体渲染出内容 | 884 / 958（约 90.4%） | **954 / 958（99.5%）** |
| Parse 失败 | 67（全部因 `encoding="ASCII"` 声明） | **0** |
| D 批成功率 | 73.4% | **99.2%** |
| C 批成功率 | 98.4% | **100%** |
| B 批全透明 | 2（含 `folder-off.svg`） | 1（仅剩真正的空 SVG） |

## 剩余 4 个全透明样例（已逐个定位，非崩溃）

| 文件 | 原因 | 归类 |
|---|---|---|
| `empty.svg` | 文件本身就是空 `<svg>`，无子元素 | 语料正常 |
| `annotation-noicon.svg` ×2 | 名为 noicon 的空图标 | 语料正常 |
| `lock-t_gZZsqx.svg` | Figma 导出：`<pattern patternContentUnits="objectBoundingBox">` + `<use>` + base64 `<image>` | **能力缺口：`<pattern>`** |

## 诊断事件分布（A 批，修复后）

| 次数 | 事件 | 结论 |
|---|---|---|
| 125 | `gradient reference not found` | **语料自身问题**：集中在 2 个 flag-icons 文件，其中一个 140KB 却 0 个 `defs`/`gradient`/`mask`，引用了不存在的 id。gosvg 的索引是全树深度遍历，行为符合规范 |
| 68 | `mask reference not found` | 同上 |
| 37 | `unsupported element skipped: foreignobject` | 已声明的子集边界；内容全是 Figma 的 `backdrop-filter:blur()` 毛玻璃 div，主体图形不受影响 |

## 复现

```sh
go run ./cmd/corpus -size 64 <dir>...     # 文本报告
go run ./cmd/corpus -json  <dir>...       # JSON，便于做基线 diff
```

语料在 .15 本机（不在仓库内），因此**该基线不进 CI**，作为手动回归手段；
CI 内的回归由 `testdata/golden.sha256` 与单元/模糊测试承担。

## 更新方式

改动渲染语义后重跑上表四批语料，若「渲染出内容」比例下降或出现新的全透明样例，
需在提交信息里说明原因；若上升，更新本文件并在 CHANGELOG 记一笔。
