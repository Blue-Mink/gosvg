# cairosvg 批量对账报告

`cmd/corpus` 的成功率指标只能回答「有没有画出东西」，回答不了「画得对不对」。
本报告用 **cairosvg 2.9.1** 作为参照实现，对真实语料做 alpha IoU 对账。

## 方法

- 语料：从 .15 上四批真实 SVG（共 958 个）按批等间隔抽样 **117 个**
- 双方均渲染为 **256×256** PNG；gosvg 用 `cmd/verify`，cairosvg 用 `svg2png(output_width/height=256)`
- 指标：alpha IoU（阈值 `alpha > 8` 视为不透明），另记双方不透明像素数与重叠区 RGB 平均差

## 结果（transform + clip/mask 顺序修复之后）

| 指标 | 值 |
|---|---|
| 样本数 | 117（cairosvg 侧 0 失败） |
| **中位 IoU** | **0.9982** |
| 平均 IoU | 0.8266 |
| 最低 IoU | 0.0476 |

分布：

| IoU 区间 | 文件数 |
|---|---|
| ≥ 0.99 | 85 |
| 0.95 – 0.99 | 4 |
| 0.80 – 0.95 | 3 |
| 0.50 – 0.80 | 2 |
| < 0.50 | 23 |

按批次：

| 批 | 语料 | n | 平均 IoU | 最低 |
|---|---|---|---|---|
| A | fnOS 面板图标 | 27 | **0.9947** | 0.9100 |
| B | node_modules 图标库 | 30 | 0.9542 | 0.0594 |
| C | 应用配置 SVG | 30 | **0.9966** | 0.9746 |
| D | docker overlay 混合来源 | 30 | 0.3776 | 0.0476 |

## 本轮修复：clip-path / mask 的求值顺序

SVG 规定 `clip-path` / `mask` 定义在「引用该属性的元素的 user space」，即元素自身
`transform` 生效**之后**的坐标系。gosvg 此前在 `renderNode` 里先算 clip/mask、
到 `renderNodeBare` 才应用 transform，**顺序反了**。

修复效果（同一次对账的前后对比）：

| 指标 | 修复前 | 修复后 |
|---|---|---|
| `A__W0PWLK5pDu-h1C6f`（flag-icons 阿鲁巴国旗，`<g transform="scale(2.2222)" clip-path=...>`） | **0.2060** | **1.0000** |
| A 批平均 IoU | 0.9653 | **0.9947** |
| 整体平均 IoU | 0.8198 | 0.8266 |
| IoU < 0.5 的文件数 | 24 | 23 |
| **退化文件数** | — | **0** |

## 剩余低 IoU 的根因分类

**D 批（数位板设备图标，IoU 0.048–0.19，共 23 个中的绝大多数）——属已知边界，非 bug：**

- 根 `<svg style="stroke:#7f7f7f;fill:none;stroke-width:.25;font-size:8">`，
  **无 `viewBox`**、只有 `width="410" height="264"`，被强制渲染成 256×256 时是非等比拉伸
- `stroke-width: .25` 经缩放后约 **0.16 px 的亚像素细线**，AA 相位差异被放大
  （与已定案的「45° 斜线 AA 相位歧义」同源）
- 含 `<text>` 标签：gosvg 用内置 Go Regular，cairosvg 用系统字体 → 字形不同
  （可用 `AddFontDir` 缓解，但会引入「渲染结果依赖系统字体」的边界）
- 佐证：这些文件的**重叠区 RGB 平均差只有 3–7**（颜色是对的），且双方不透明像素数接近
  —— 是「画在同一处但 AA/字形不同」，不是语义错误

**B 批 `frame-icon.svg`（IoU 0.0594，gosvg 1464 vs cairosvg 24562 不透明像素）——根因待查：**

- Inkscape 导出，仅 5 个元素、1 条 `fill-rule="evenodd"` 的 path
- `<g transform="matrix(0.26458333,0,0,0.26458333,-0.066,-0.397)">`
- gosvg 画出的像素远少于 cairosvg，**疑似真缺口**（evenodd + 复杂 path 数据，或
  `H`/`V`/`c`/`v` 相对指令与紧凑小数组合），已列入待办

## 复现

```sh
# 1) .15 上用 gosvg 渲染抽样目录
go run ./cmd/verify <in_dir> <out_dir>          # 256x256 PNG
# 2) 本机（需 cairosvg）渲染同一批
python3 -c "import cairosvg,glob,os;[cairosvg.svg2png(url=f,write_to='cairo/'+os.path.basename(f)[:-4]+'.png',output_width=256,output_height=256) for f in glob.glob('<in_dir>/*.svg')]"
# 3) 用 PIL/numpy 按 alpha>8 算 IoU
```

> 对账依赖 cairosvg，CI 环境没有，因此**不进 CI**；作为发版前的手动质量关口。
> 语料本身在 .15 上（不在仓库内），抽样口径见「方法」一节。

## 结论

1. **中位 IoU 0.9982**：对绝大多数真实 SVG，gosvg 与 cairosvg 几乎逐像素一致。
2. 平均值被 D 批拉低，而 D 批的差异经逐项核查是**亚像素细线 AA + 字体**，属已定案边界。
3. 本轮靠对账定位并修复了一个**成功率指标完全看不到的真 bug**（clip/mask 与 transform
   的求值顺序），单文件 IoU 0.206 → 1.0000、零退化 —— 这验证了「对账比频率统计更能
   定位真实缺口」的方法论。
4. 下一个待查目标是 B 批 `frame-icon.svg`（0.0594）。
