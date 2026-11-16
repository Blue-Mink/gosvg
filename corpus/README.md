# 850 语料 IoU 对账记录（仓库外语料）

本目录固化 gosvg 对 **850 个真实 SVG** 做 alpha IoU 对账的**逐样本数字**与**重跑工具**。
原始 SVG / 参照 PNG **不在仓库内**（见下方「为何原始资产不入仓」），因此本目录**不进 CI**；
它与 `testdata/golden.sha256`（仓库内样例像素哈希、防无意改动渲染）是两套互补机制。

## 文件

| 路径 | 内容 | 口径 |
|---|---|---|
| `corpus/ious/iou_withfont_a8.json` | 848 样本逐项 IoU | 本机带字体复现、alpha 阈值 8 |
| `corpus/ious/iou_bothfont_a0.json` | 848 样本逐项 IoU | 本机双字体复现、alpha 阈值 0 |
| `corpus/scripts/render_ref.py` | cairosvg 参照图生成（通用版） | 256x256、透明背景 |

每个 JSON 的 `samples[]` 记录 `name`（去扩展名 stem）、`iou`、`ref_pixels`、`got_pixels`，
汇总字段有 `compared` / `missing` / `average_iou` / `below_0.5`。**只有数字与文件名，无任何图像内容。**

## 口径：不可与文档基线横比

这两个 JSON 是 **2026-11-15/16 本机复现与证伪实验**的记录，**不是权威基线**：

| 来源 | average IoU | 说明 |
|---|---|---|
| `iou_withfont_a8.json` | 0.951315 | 本机隔离字体 + alpha>8 |
| `iou_bothfont_a0.json` | 0.950931 | 本机双字体 + alpha>0；未复现历史 0.960035，A 方案已证伪 |
| `CAIROSVG_IOU.md`（v0.2.3, alpha>0） | 0.963860 | 历史权威口径 |
| `CHANGELOG.md`（.15 有 54 字体） | 0.960035 | 历史权威口径，原始逐项 JSON 已丢失 |

本机 cairosvg 字体环境与 .15 不同（本机字体数为 0，隔离 54 字体后仍未复现），
加之历史 `cairo_all` 参照图原件已丢失，**这些数字与文档基线口径不同、不可横比**。
保留它们是为记录「每个样本在当前本机环境的表现」，供复算与回归对照，而非宣称新基线。

## 语料来源与版权

原始 850 SVG 来自 .15 本机真实语料，按批次：

| 批次 | 本语料数 | 来源 | 许可 |
|---|---|---|---|
| A | 187 | fnOS 面板图标 `/usr/trim/www`，含 flag-icons | flag-icons MIT；其余专有 |
| B | 308 | node_modules 图标库（Lucide 等） | ISC/MIT/专有混杂 |
| C | 127 | 应用配置 SVG | 应用自带 |
| D | 228 | docker overlay 混合来源 | 来源不明 |

其中 **48 个文件含 copyright/license/source 注释**。原始资产版权归属混杂、部分专有或来源不明，
**再分发有法律风险**，故不入库。文件名已脱敏为 `批次__随机串.svg`，但内容仍是原图。

## 为何原始资产不入仓

1. **版权**：见上，48 个含版权标记，A/B/D 批含专有或来源不明资产，不宜公开发布。
2. **体积**：850 SVG（9.8M）+ 848 参照 PNG（7.3M）约 17M，会让 2.5M 的源码仓库膨胀 7 倍，
   历史永久携带约 1700 个二进制对象。
3. **项目既定决策**：`CORPUS_BASELINE.md` 与 `CAIROSVG_IOU.md` 均已写明「语料在 .15 本机、不在仓库内、不进 CI」。
4. **.gitignore**：明确忽略 `*.png`，项目本就不打算把参照 PNG 入仓。

## 如何复算

在持有原始语料的机器上：

```sh
# 1) 生成参照图（需 cairosvg 2.9.1；带字体口径另传 --fontconfig）
python3 corpus/scripts/render_ref.py --svg <SVG目录> --ref <参照目录>

# 2) 用 gosvg 渲染并与参照图对账，输出与上面同格式的 JSON
go run ./cmd/iou -svg <SVG目录> -ref <参照目录> -json <结果.json> -size 256

# 3) 全量校验（含真 -race）
sh ./scripts/verify.sh
```
