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


---

## 全量复测（848 个样本）与上一版结论的修正

> **重要：本节推翻了上面「剩余低 IoU 的根因分类」里对 D 批的判断。**

把对账从 117 抽样扩到 **848 个样本**（全量 958 个里去掉跨批次重名 108 个、cairosvg 渲染失败 2 个）。结果：

| 指标 | 抽样 117 | 全量 848 |
|---|---|---|
| 中位 IoU | 0.9982 | 0.9978 |
| 平均 IoU | 0.8266 | 0.7870 |
| IoU < 0.5 | 23 | **210** |

按批次（全量，修复前）：

| 批 | n | 平均 IoU | 中位 | <0.5 |
|---|---|---|---|---|
| A | 187 | 0.9905 | 1.0000 | 0 |
| B | 308 | 0.9709 | 0.9993 | 9 |
| C | 127 | 0.9558 | 0.9978 | 5 |
| **D** | **226** | **0.2731** | **0.1800** | **196** |

### 修正 1：D 批不是「亚像素 AA + 字体」

上一版把 D 批判为已定案边界，依据是「重叠区 RGB 差仅 3–7」。**这个推理是错的**：位置/尺度错开时重叠区趋近于零，该指标天然失真（IoU=0 时它甚至是 0）。

实证：`B__resizeHorizontal.svg` 是 `<svg width="6" height="26">`（无 viewBox、6:26 极端长宽比）：

```
gosvg  bbox=(0,0,255,255)    像素 43367   ← 非等比拉伸填满正方形
cairo  bbox=(98,0,157,255)   像素 10577   ← 等比缩放 + 水平居中（59×256）
```

cairo 的 59 = 6 × min(256/6, 256/26)，居中偏移 98.5。根因是 `render.go` 的
`viewBoxMatrix` 在 `!hasVB` 分支直接 `Scale(w/vw, h/vh)` **非等比拉伸**。
D 批 228 个里 **196 个无 viewBox**，bbox 的 Y 方向系统性错位；`stroke-width:.25`、
`<text>` 字体确实存在，但只是叠加因素，不是主因。

**修复**：无 viewBox 时按 `width/height` 等比缩放并 `xMidYMid` 居中；
`preserveAspectRatio="none"` 才保留拉伸。6 个 golden 样例全有 viewBox，基线不受影响。

**全量复测（848 样本）**：

| 指标 | 修复前 | 修复后 |
|---|---|---|
| 平均 IoU | 0.7870 | **0.9486** |
| IoU < 0.5 | 210 | **17** |
| D 批平均 IoU | 0.2731 | **0.8687**（<0.5：196 → 5） |
| 提升 > 0.05 的文件 | — | **195 个** |
| **退化 > 0.05 的文件** | — | **0 个** |

`resizeHorizontal` / `resizeVertical`：**0.0000 → 0.999**。

### 修正 2：even-odd + transform 假设被证伪

曾怀疑 `clip-path`/`mask` 之外还有 even-odd winding 与缩放 transform 组合的 bug
（`internal/vector` 用 `ceil(|acc|/unit)` 从累积面积反推圈数，边缘非对齐时会误判；
`B__frame-icon.svg` 是个例：nonzero 24582 ≈ cairosvg 24562，evenodd 只有 1480）。

全量交叉验证后**证伪**：

| | 数量 |
|---|---|
| 声明 `fill-rule=evenodd` 的文件 | 171 |
| 其中含多子路径 | 163 |
| 多子路径 + 非整数缩放/平移 transform（高危） | **70** |
| 高危组 IoU < 0.5 | **3** |
| 高危组平均 / 中位 IoU | 0.9434 / **1.0000** |
| IoU < 0.9 的 224 个里命中高危 | 6 |

高危组反而比非高危组好（中位 1.0000 vs 0.9978）。`frame-icon` 是个例，
**不值得为此改 `internal/vector` 核心**。已撤回该修复计划，只留作待查项。

### 修复后剩余 72 个 IoU < 0.8 的构成

| 批 | 数量 | 新根因线索 |
|---|---|---|
| D | 54 | `width="193mm" height="180mm"` **物理单位未换算**（1mm≈3.78px），视口分母错；叠加亚像素细线与字体 |
| B | 9 | `clip-path` 与子元素 `transform` 组合（`frame-icon` 0.0594、`resizeDiagonal` 0.0674） |
| C | 7 | `width/height="1em"`、evenodd 个例（bbox 相同但只画出 ~6% 像素，如 `1f9fab4e48601ca5` 0.0580） |
| A | 2 | 待查 |

**下一个目标**：先修「width/height 带物理单位（mm/cm/in/pt）未换算」——它影响的 D 批文件最多
（54 个），且 `parseLength` 目前只是剥掉单位字母，属于当初明确决策「单位 1:1 近似」
留下的洞。注意这与「长度单位真换算会改变渲染字节与 IoU 表」的旧顾虑不同：
这里换算的是**视口尺寸**，属于视口建立语义，不是元素几何。

### 方法论（再次校准）

1. **抽样会骗人**：117 抽样里 D 批只占 30 个、低分被当作个例；全量 848 才暴露 196 个。
2. **「成功率/像素数」不能代替几何对比**：`resizeHorizontal` 在 `cmd/corpus` 里
   100% 渲染出内容，但 IoU = 0（画在了完全不同的位置）。
3. **bbox 对比比 RGB 差可靠**：位置错时 RGB 差会假性变小。
4. 假设（even-odd）必须做**全量交叉验证**再动手，否则会去改不该改的 fork 核心。
