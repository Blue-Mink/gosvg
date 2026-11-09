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

## 结论（修正 1 阶段；**第 2 / 4 条已被后续修正推翻，见节内标注**）

1. **中位 IoU 0.9982**：对绝大多数真实 SVG，gosvg 与 cairosvg 几乎逐像素一致。
2. 平均值被 D 批拉低，而 D 批的差异经逐项核查是**亚像素细线 AA + 字体**，属已定案边界。
   > ⚠️ **已废弃**：修正 2 证明 D 批低分的主因是**无 viewBox 的非等比拉伸**，不是 AA/字体；
   > 修正 3 又证明物理单位换算也不是剩余低分的主因。以修正 2 / 3 的结论为准。
3. 本轮靠对账定位并修复了一个**成功率指标完全看不到的真 bug**（clip/mask 与 transform
   的求值顺序），单文件 IoU 0.206 → 1.0000、零退化 —— 这验证了「对账比频率统计更能
   定位真实缺口」的方法论。
4. 下一个待查目标是 B 批 `frame-icon.svg`（0.0594）。
   > ⚠️ **已废弃**：修正 2 的全量交叉验证显示 even-odd + transform 假设被证伪、
   > `frame-icon` 是个例；当前真实待查项见修正 3 末尾列出的 2 个整块缺失文件。


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

### 视口换算修完后剩余 70 个 IoU < 0.8 的构成

| 批 | 数量 | 根因线索 |
|---|---|---|
| D | **52** | 50 个为抗锯齿 / 字体栅格化边界（bbox 与像素数都近似）+ 2 个填充差异；**不再是物理单位问题** |
| B | 9 | `clip-path` 与子元素 `transform` 组合（`frame-icon` 0.0594、`resizeDiagonal` 0.0674） |
| C | 7 | `width/height="1em"`、evenodd 个例（bbox 相同但只画出 ~6% 像素，如 `1f9fab4e48601ca5` 0.0580） |
| A | 2 | 待查 |

### 修正 3：视口物理单位换算（已落地，但收益很小）

按上一节列出的「下一个目标」实施：新增 `viewport.go` 的 `parseViewportLength`，把根
`<svg>` 的 `width`/`height` 按 **96dpi** 换算 `px/mm/cm/in/pt/pc/q`，`parser.go`
的 `d.width` / `d.height` 改走它。元素几何（`font-size` / `stroke-width` 等）仍走
`parseLengthOr` 的 1:1 近似旧决策 —— 换算的是**视口建立语义**，不是元素几何。

**全量 848 样本复测**：

| 指标 | 修复前 | 修复后 |
|---|---|---|
| 平均 IoU | 0.9486 | **0.9507** |
| D 批低分文件 | 54 | **52** |
| 退化 > 0.05 的文件 | — | **0** |
| 提升显著的文件 | — | **2 个 bamboo**（`0.0000 → 0.9334` / `0.8604`） |

**结论：物理单位换算不是那 52 个低分的原因。** 假设被全量数据证伪，与修正 2
（even-odd / transform）一样如实记录，不为了「让数字好看」继续改内核。

剩余 52 个逐项分类（bbox 与不透明像素数双指标核对）：

| 类别 | 数量 | 判据 |
|---|---|---|
| 抗锯齿 / 字体栅格化边界 | **50** | 双方 bbox 与不透明像素数都近似，IoU 落在 0.70–0.77；差异只在边缘像素覆盖与字形栅格化，属渲染器级像素差，不是几何错误 |
| **整块缺失（真 bug）** | **2** | `toolbarButton-editorHighlight` 0.40、`altText_spinner` 0.45 |

**重新定性（PNG 逐像素复核，推翻上一版「填充差异 / 低优先级」的判断）**：这两个不是
局部填充色差，而是 gosvg **整块图形没画出来**——

| 文件 | gosvg 不透明像素 | cairosvg | gosvg 多画 | cairosvg 多画 | 形状对比 |
|---|---|---|---|---|---|
| `toolbarButton-editorHighlight` | 8486 | 20701 | **5** | **12220** | gosvg 只剩 even-odd 多子路径的空心轮廓，cairosvg 是实心笔身 |
| `altText_spinner` | 15585 | 34241 | **5** | **18661** | gosvg 缺外圈与中间一块，cairosvg 两个元素都画 |

根因假设（只读定位，未改代码）：

- `altText_spinner`：`<style>` 里的 `@media` 是嵌套块，而 `parseCSSLimited` 按 `}` 切块、
  按 `{` 拆分，会把块内的 `#hourglass { display: none; }` **当成顶层规则应用**，整块被隐藏。
  规范上 `@media` 属「明确不支持」应整块忽略，而非部分生效。
- `toolbarButton-editorHighlight`：单条 path 含 3 个 `Z` 子路径且 `fill-rule="evenodd"`，
  gosvg 的 even-odd winding 把它算成近似全挖空；cairosvg 画成实心。

判定：**50 个 AA/字体边界归入已定案边界，不值得继续改**；这 2 个是真 bug，但修它们会
改变渲染字节与全量 848 样本 IoU 表、需要重跑对账与 golden 基线，按纪律列为
**需用户点头**项，不擅自改、不阻塞发版。修正 3 的价值主要是**语义正确性**
（与浏览器 / cairosvg 的视口建立一致），而非 IoU 提升。

### 修正 5：even-odd 修复假设证伪（2026-11-09，全量验证后撤回）

> 编号说明：本结论在实验分支 `chore/evenodd-scan-finding`（已关闭的 PR #13）中记为
> 「修正 5」；其前序「修正 4」（even-odd 状态残留假设与该假设下的修复尝试）从未合入 main，
> 故本文件编号由 3 直接跳至 5，属有意保留，以免与分支记录、PR 讨论产生歧义。

**结论：PR #13 / `9dd24ac` 的「独立 Filler 实例」改动是 no-op，对声称要修的缺陷没有任何可见效果；此前「共享 `ctx.dasher.Filler` 导致 even-odd 状态在多次 fill 之间残留」这一根因假设不成立。修复撤回，PR 已关闭、未合并、未发版。**

#### 验证（本机自建 Go 1.27.3，与 .15 同版本；未触碰 .15）

- `sh ./scripts/verify.sh` → **ALL GREEN**：gofmt 空、vet 0、test 86.2%、`-race` 全过、arm64/wasm 交叉编译、bench、FuzzParse 30s + FuzzRender 20s 无 crash。
- `testdata/golden.sha256` **未变**（6 个基线样本渲染字节未变），无需重生成。
- 全量 850 个 SVG 重新渲染（零错误），与修复前基线及 cairosvg 参照按同一 alpha IoU（alpha>8）逐文件比对，**848 个可比样本**：

  | 指标 | 修复前 | 修复后 |
  |---|---|---|
  | 平均 IoU | 0.9507 | 0.9507 |
  | 中位 IoU | 0.9978 | 0.9978 |
  | IoU < 0.5 | 15 | 15 |
  | IoU < 0.8 | 70 | 70 |

  逐文件 Δ：提升 >0.001 = **0**，退化 <-0.001 = **0**，|Δ| 最大 **0.0000**，848 个**逐像素完全相同**。真 bug 文件 `D__toolbarButton-editorHighlight` 修复后仍为 **0.4048**，`D__altText_spinner` 仍为 **0.4547**。

- 严格交叉验证（排除基线不干净）：从 `git archive 7bf4905`（main，修复前）单独编译二进制渲染同一文件，四方像素 sha256 与不透明像素数完全一致：

  | 来源 | RGBA sha256(前 16) | alpha>8 像素 |
  |---|---|---|
  | baseline(7bf4905) / new(9dd24ac) / 历史基线 / 本轮全量 | `915deee870f66710` | 8343 |
  | cairosvg 参照 | `5f412c4682b3ba8e` | 20580 |

#### 为什么无效

`internal/rasterx/scan.go:71` `SetWinding(b)` 仅做 `s.r.UseEvenOdd = !b`；而 `render.go fill()` **每次调用开头都会执行 `rf.SetWinding(nonZero)`**（`nonZero = style.fillRule != "evenodd"`），原代码末尾还有 `rf.SetWinding(true)` 复位。因此「共享 Filler 导致状态残留」的假设**根本不成立**——改成每次新建 Filler 与原逻辑等价，渲染字节必然不变。verify 全绿、golden 通过恰恰是因为**渲染字节压根没变**，不能当作「修复生效」的证据。

真正缺陷（`toolbarButton-editorHighlight`：单条 path 含 3 个 `Z` 子路径 + `fill-rule="evenodd"`，gosvg 8343 像素空心轮廓 vs cairosvg 20580 像素实心）出在 **rasterx 对多子路径的 even-odd 奇偶累积算法**层面，与 Filler 实例生命周期无关；`altText_spinner` 主因仍是 CSS `@media` 解析。两者均未在本次改动中触及。

#### 决定与现状

- PR #13 **已关闭、未合并**（标题/正文含「已修复」的不实结论，合并会把错误结论写进 main）。
- 本次仅追加文档结论：渲染代码、`golden.sha256`、全量 IoU 基线均未改动；未打 tag、未发 release。实验分支上曾列出的后续修复/发版计划与环境阻塞说明，随本次撤回一并作废（环境阻塞已通过本机自建工具链解决）。
- 若日后要真修，需另行授权：动 `internal/rasterx` 多子路径 even-odd 算法，或改走 SVG 预处理等价转换；之后必须重跑 `verify.sh` 全套 + 全量 IoU **生效验证** + 更新 `golden.sha256`。当前缺陷收益极小（15 个低分中的 2 个），按既有定调不值得为此改 fork 核心。

### 方法论（再次校准）

1. **抽样会骗人**：117 抽样里 D 批只占 30 个、低分被当作个例；全量 848 才暴露 196 个。
2. **「成功率/像素数」不能代替几何对比**：`resizeHorizontal` 在 `cmd/corpus` 里
   100% 渲染出内容，但 IoU = 0（画在了完全不同的位置）。
3. **bbox 对比比 RGB 差可靠**：位置错时 RGB 差会假性变小。
4. 假设（even-odd）必须做**全量交叉验证**再动手，否则会去改不该改的 fork 核心。
5. **「ALL GREEN + golden 通过」不等于「修复生效」**：无退化验证与生效验证是两件事。改动渲染代码后，除 `verify.sh` 外必须做修复前后**逐文件 IoU / 像素差异**比对，确认目标文件真的动了、且没有误伤；本次正是靠这一步拦下了一个看似完整、实则零效果的 PR。
