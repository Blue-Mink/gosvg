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

> ⚠️ **「个例」定性已被修正 6 的实测推翻**：全量 848 重算后，15 个 IoU<0.5 里有
> **11 个**是同类 even-odd 缺陷（跨 B 6 / C 4 / D 1 三批），`frame-icon` 只是其中之一；
> 把它归为「transform 分组误判」也不成立，它与 transform 无关。
> 「不改 `internal/vector` 核心」的结论仍然成立，但「个例 / 只留待查项」已作废。

### 视口换算修完后剩余 70 个 IoU < 0.8 的构成

| 批 | 数量 | 根因线索 |
|---|---|---|
| D | **52** | 50 个为抗锯齿 / 字体栅格化边界（bbox 与像素数都近似）+ 2 个填充差异；**不再是物理单位问题** |
| B | 9 | `clip-path` 与子元素 `transform` 组合（`frame-icon` 0.0594、`resizeDiagonal` 0.0674） |
| C | 7 | `width/height="1em"`、evenodd 个例（bbox 相同但只画出 ~6% 像素，如 `1f9fab4e48601ca5` 0.0580） |
| A | 2 | 待查 |

> ⚠️ **本表两处定性已被修正 6 推翻**：D 批「50 个 AA/字体边界 + 2 个填充差异」与
> C 批「evenodd 个例」均不成立。用「双方 bbox 与不透明像素数都近似」判 AA 会漏掉
> 内部几何错误；even-odd 缺陷实测影响 11 个低分文件，相对全量富集约 27 倍，不是个例。

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

> ⚠️ **该判据已被修正 6 证伪**：「bbox 与不透明像素数都近似」只能排除位置 / 尺寸错误，
> **排除不了「内部被挖空」**。反例 `B__issue-exclamation-filled`：gosvg 9142 / cairosvg 9155
> 不透明像素、bbox 完全相同，IoU 却只有 **0.3497** —— 感叹号整块被 even-odd 挖掉。
> 据此得出的 50 / 2 划分与「2 个真 bug」计数均不作数。

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

> ⚠️ **「2 个真 bug / 不值得改」已被修正 6 推翻**：真实影响面是 15 个低分中的 **11 个**，
> 加上 19 个此前未计入低分账的中低分文件，共 **30 个文件**受益；PoC 实测零退化。
> 但「不擅自改渲染代码、需用户点头」的纪律仍然有效，本次也确实没有改。

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

  > ⚠️ **上一行「当前缺陷收益极小（15 个低分中的 2 个）」已被修正 6 的实测推翻**：
  > 影响面是 15 个低分中的 **11 个**（富集约 27 倍）；PoC 证明「SVG 预处理等价转换」可把
  > 全量平均 IoU 从 **0.950746 提到 0.962169（+0.011423）**且 61 个改动文件零退化。
  > 本节「`9dd24ac` 为 no-op」的证伪结论不受影响，仍然成立。

### 修正 6：even-odd 影响账重算 + SVG 预处理等价转换 PoC（2026-11-09，全量回归零退化）

> 本节**推翻修正 2 / 3 / 5 中「个例」「50 个 AA/字体边界」「15 个低分中的 2 个」「收益极小」四处定性**。
> 历史段落按原样保留、仅在段后加注引用本节，不抹掉当时的判断。修正 5 关于 `9dd24ac` 为 no-op 的证伪结论**仍然成立**。

#### 影响账重算

| 指标 | 数值 |
|---|---|
| 全量可比样本 | 848 |
| IoU < 0.5 | 15 |
| 其中同时声明 `fill-rule="evenodd"` + `clip-rule="evenodd"` | **11** |
| 全量声明该双规则的文件 | 61 |
| 命中率 | 低分组 11/15 = 73.3% vs 全量 61/848 = 7.2%，**富集约 27 倍**（Fisher 精确检验 p ≈ 1e-9） |
| 跨批次分布 | B 6 / C 4 / D 1，非单批伪影 |

4 组**单属性消融 A/B**（每次只改一个属性、不改代码，用同一 main 基线二进制渲染）：

| 变体 | 11 文件平均 IoU | 结论 |
|---|---|---|
| 原版 | 0.2533 | — |
| 只把 `clip-rule` 改成 `nonzero` | 0.2533（**逐位相同**） | 裁剪规则无关 |
| 只把 `fill-rule` 改成 `nonzero` | 0.9005 | **填充规则是唯一开关** |
| 两者都改 | 0.9005 | 同上 |

`clip-rule` 逐位无变化，说明缺陷在**填充光栅化**，不在裁剪路径。

#### 根因收窄

- 最小复现与对照（`rect_hole_evenodd`、`two_subpaths_evenodd`、`frame-icon` false-positive、`two_disjoint_evenodd`、`evenodd_presubpath_state` 等）排除了「多子路径 even-odd 累积算法整体坏」；触发条件收窄为 **单条 `<path>` 内含 2–4 个 `Z` 子路径 + `fill-rule="evenodd"`**。
- 修正 2 把 `frame-icon` 归为「transform 分组误判」，实测证伪：它就是 even-odd 缺陷，与 transform 无关。
- `D__altText_spinner` 主因仍是 CSS `@media` 嵌套块解析，与本节无关。
- 11 个文件的 `d` 指令集均为 `M/L/H/V/C/Z` 及相对形式，**无弧线 `A`**，子路径结构都是「外框 + 内孔」或「外圈 + 内圈」。

#### PoC：SVG 预处理等价转换

思路：把 `fill-rule="evenodd"` 的多子路径 `<path>`，按嵌套深度把子路径方向调整为**相邻层级相反**，再改走 `fill-rule="nonzero"`，**保持单条 `<path>`**。关键约束是「单条」——拆成多条独立 `<path>` 会各自 fill、后画的覆盖先画的，winding 的跨子路径累积丢失，根本无法表达挖空。

| 变体 | 平均 IoU | 说明 |
|---|---|---|
| 原版 | 0.2533（11/11） | — |
| `single_eo`（重建但不调方向、仍 evenodd） | **0.2533，逐位相同**（11/11） | 对照：证明转换流程无损、无精度误差 |
| `split_eo`（拆成多条 path，各自 evenodd） | 0.2058（10/11） | 证伪「只拆不调方向」 |
| `nonzero`（拆成多条 path + 交替方向） | 0.7817（10/11） | **劣于**直接改 fill-rule 的 0.9005，证伪「拆 path」这条路 |
| **`single_nz`（单 path + 交替方向 + nonzero）** | **0.9924（11/11）** | 逐个文件 0.9665–0.9998 |

> 两种「拆成多条 path」的变体各有 1 个文件因生成器把非自闭合 `<path></path>` 写坏而未渲染，
> 故按 10 个有效值平均；该生成器 bug 已在最终方案中修复，`single_nz` 11/11 全有效。
> 分母差异不改变结论：拆 path 的两组都明显劣于直接改 `fill-rule` 的 0.9005。
> （PR #15 初稿曾把拆分组按 11 作分母写成 0.7107，此处更正。）

方向判定规则（而非无条件反转「奇数深度」）：取深度 0 中子路径的符号为基准，深度 d 的目标符号为 `基准 × (-1)^d`，当前符号不符才反转。多数文件的子路径方向**本来就交替**（外 `+` / 内 `−`），无条件反转会把它们弄成同向 —— 这正是第一轮 PoC 里 `frame-icon` 掉到 0.5207 的原因。

#### 全量回归（保守 scope）

只转换「指令集受支持、无 `<clipPath>`、多子路径」的 evenodd path；含 `A`/`T` 弧线指令与 clipPath 内的一律**原样跳过**，未转换文件逐字节不变。

| | 数量 |
|---|---|
| 全量 SVG | 848 |
| 逐字节未改动 | **787**（680 无转换 + 107 含 clipPath 跳过） |
| 实际改动 | **61**（158 条 path） |
| 被跳过的含 `A`/`T` path | 260 |
| 改动文件回归 | **30 提升 / 31 不变 / 0 退化** |
| 不变的 31 个 IoU 区间 | 0.9962–1.0000 |
| 全量平均 IoU | **0.950746 → 0.962169（+0.011423）** |

除 11 个低分文件外，还有 **19 个此前未计入「低分账」的文件**也提升，如 `C__099fe453ccedf43a` 0.6334 → 0.9991 —— 影响面比「11 个」更宽。

#### 现状与边界

- **PoC 全部由执行机 `/tmp/evenodd_poc/` 下的外部脚本完成**：未改 gosvg 仓库任何文件、未动 `internal/rasterx` / `render.go` / `gradient.go`、未重跑 `verify.sh`、未更新 `golden.sha256`、未打 tag、未发 release、未碰 .15。脚本与数据未入库。
- 等价转换**不是完全通用**：对自交子路径、部分重叠（非嵌套）子路径，even-odd 与「按嵌套深度交替方向的 nonzero」不等价；嵌套判断用代表点 + 面积比较，复杂形状可能误判。落地前需补这些 case 的测试。
- 若要落地，需在解析 / 渲染层加一个 Go 侧等价转换（**不动光栅器核心**），之后仍需 `verify.sh` 全套 + 全量 IoU 生效 / 误伤验证 + `golden.sha256` 重生成。**需用户单独授权，本次未做。**

  > ✅ **已由修正 7 落地**（2026-11-10，用户回复「按你的建议来」授权）：转换在解析层实现、
  > 光栅器核心零改动；`verify.sh` 全套 ALL GREEN；全量 848 样本生效 / 误伤验证完成。
  > `golden.sha256` 经核验**无需重生成**（6 个确定性样例均不含 `fill-rule` 声明）。

### 修正 7：even-odd 等价转换的 Go 侧落地与全量生效验证（2026-11-10）

用户回复「按你的建议来」授权后，把修正 6 的外部 SVG 预处理 PoC 落成库内实现。
**边界与建议一致**：只在解析 / 渲染层做转换，`internal/rasterx` 与 `internal/vector`
一行未改；描边与 marker 仍用原始 path，只有填充路径被替换。

#### 实现

| 文件 | 改动 |
|---|---|
| `evenodd_convert.go`（新增） | 在已归一化的 `rasterx.Path` 上按 `PathMoveTo` 切子路径，采样折线算 shoelace 有向面积与嵌套深度，按需反转段序列（控制点倒序、终点换成原段起点、`Z` 隐含段补成显式 `Line`），重建一条「方向交替、走 nonzero」的等价 Path |
| `render.go` | `renderShape` 里把填充改为 `fillPath` + `fillNonZero`；`inClipPath` 标志排除 clipPath 内部路径 |
| `clip.go` / `mask.go` / `pattern.go` | 创建离屏 `renderContext` 时传递 / 设置 `inClipPath`，避免嵌套离屏渲染漏过排除条件 |
| `evenodd_convert_test.go`（新增） | 10 项测试：同向嵌套、三层 winding、独立子路径、三次贝塞尔反转控制点顺序、方向交替断言、gradient 填充、clipPath 跳过、A/T 跳过、Path 结构合法性，以及**自交 / 部分重叠两类拒绝用例** |

**保守条件**（任一不满足即放弃转换，回退原 even-odd 光栅化，宁可保留已知缺陷也不引入新错误填充）：
`style.fillRule == "evenodd"`、不在 clipPath 内、`d` 不含 `A/a/T/t`、至少 2 个子路径、
无自交、无边界互相穿越（部分重叠）、无零面积子路径、采样总点数 ≤ 512。
这正面回应了修正 6「落地前需补自交 / 部分重叠 case 测试」的遗留项 —— 不是只补测试，
而是让实现**检测并拒绝**这两类不等价输入。

#### 验证

- `sh ./scripts/verify.sh` **ALL GREEN**（gofmt / vet / test -cover / `-race` / arm64 /
  wasm / bench smoke / FuzzParse 30s / FuzzRender 20s）。`testdata/golden.sha256` 未变，
  经查 6 个样例 `blues` / `cn` / `country` / `jazz` / `lucide` / `rock` 均无 `fill-rule`
  声明，属于真实的「无需更新」，不是漏检。
- **全量生效 / 误伤验证**：用 `git archive HEAD` 导出干净基线快照，与改动后代码各编译一个
  `cmd/render`，以相同参数（`gosvg.Render(data, 256)` + `png.Encode`）各渲染 850 个 SVG；
  按修正 6 同一口径（`alpha>0` 二值 mask 的 `交/并`）与 `cairo_all` 的 848 个参考图对比。

| 指标 | 基线 | 落地后 |
|---|---|---|
| 参与对比样本 | 848 | 848 |
| 像素发生变化的文件 | — | **30** |
| IoU 提升 / 退化 / 不变 | — | **30 / 0 / 818** |
| 全量平均 IoU | 0.949874 | **0.961272（+0.011397）** |
| IoU < 0.5 的文件数 | 15 | **4** |
| 11 个目标文件平均 IoU | 0.2560 | **0.9927** |

逐文件提升明细（30 个全部为正向，最大 `B__frame-icon` 0.0602 → **1.0000**、
`C__1f9fab4e48601ca5` 0.0620 → 0.9986、`D__toolbarButton-editorHighlight` 0.4096 → 0.9981）。
**11 个低分目标全部救回，另有 19 个此前未计入低分账的中低分文件受益**，与修正 6 的
PoC 结论（30 提升 / 31 不变 / 0 退化）一致。两套平均 IoU 绝对值略有差异（PoC 为
0.950746 → 0.962169）是因为 PoC 的输入是外部转换后的 SVG、基线渲染不同，不可逐位对比；
相对增量 +0.011397 与 +0.011423 一致。

- **剩余 4 个 IoU < 0.5 文件像素零变化**，均非 even-odd 缺陷：`B__resizeDiagonal`
  0.0677（clip-path + transform）、`C__6ab88ea05959193c` 0.2432、
  `D__wacom-intuos-pro-2-s` 0.4397、`D__altText_spinner` 0.4549（`@media` 嵌套 CSS）。
- **性能无劣化**：语料中最重的 `A__CJIHhYwFz-erht8H.svg`（77 KB / 552 个 path /
  257 个 `Z`）单文件渲染基线 22.567s、落地后 21.707s。全量首次顺序渲染超时是预算被
  基线批次占用，非转换引入的卡顿（已用单文件计时排除）。
- **未打 tag、未发 release**，发版仍需用户单独确认。


### 修正 8：剩余 4 个低分的三类结构性缺陷（2026-11-10）

修正 7 落地后全量 848 样本剩 4 个 `IoU < 0.5`。逐个做像素账 + 单属性消融 A/B 后，
其中 3 个是互不相关的结构性 bug，1 个是既有的字体 / AA 边界（定案不修）。

#### 根因与修复

| 文件 | 现象 | 根因 | 修复 |
|---|---|---|---|
| `B__resizeDiagonal` 0.0677 | 底部一条水平横条，而非两条 45° 斜线 | `render.go:renderNodeBare` 的 `skipNodeTransform` 是「同一节点重渲染时别重复应用 transform」的一次性标记，却随 `renderSubimage` 传进 sub context 并泄漏给整棵子树，`<g clip-path>` 下每个 `<rect transform="rotate(...)">` 都被吞掉 | 应用完当前节点 transform 后立即复位该标记 |
| `C__6ab88ea05959193c` 0.2432 | 1696×240 的 sprite sheet 被压成一个扁图 | `image.go:renderImage` 把**已裁剪**的 `drawRect` 直接配完整 `src` 交给 `xdraw.Scale`，等于把整图缩放进可见区 | 按 `drawRect` 在完整目标矩形中的比例切出对应 `srcRect`，再缩放 |
| `D__altText_spinner` 0.4549 | 沙漏整块消失，只剩环形箭头 | `css.go:parseCSSLimited` 按 `}` 暴力切块，`@media` 嵌套里的 `#hourglass{display:none}` 被提升为顶层规则；同块 `@keyframes` 与第二个 `@media` 又让另一条规则的选择器变成垃圾字符串 | 大括号深度感知扫描，整块跳过 at-rule（gosvg 不求值媒体查询，与 cairosvg 一致），顶层规则照常生效 |
| `D__wacom-intuos-pro-2-s` 0.4397 | 形状一致，笔画差 1 列 | 9 个 `<text>` 的字体度量 / AA 边界；图形总像素仅 537，几个像素就吃掉一半 IoU | **不修**，沿用既有定案 |

三个假设都先做了只改 SVG 的 A/B：去掉 `clip-path` 后 `B__resizeDiagonal` 0.0677 →
0.9973、去掉 `<style>` 后 `altText_spinner` 0.4549 → 0.9990，确认根因后再动代码。
另补 3 个回归测试（`cssmedia_test.go` / `transformclip_test.go` / `image_test.go`），
并验证它们在**基线代码上全部 FAIL**、修复版全 PASS，避免写出摆设测试。

#### 验证（instabox 沙箱独立环境，未触碰 .15 的 QwenPaw）

- `sh ./scripts/verify.sh` **ALL GREEN**（含 3 个新测试），覆盖率 86.1% → **86.5%**；
  `testdata/golden.sha256` 校验通过，无需重生成。
- 全量 848 样本逐文件 IoU / 像素对账（口径同修正 7）：

| 方案 | 平均 IoU | IoU<0.5 | 提升 | 退化 |
|---|---|---|---|---|
| 基线 `fb9ffee` | 0.961272 | 4 | 0 | 0 |
| 仅 transform + image 两处 | 0.963245 | 2 | 4 | **0** |
| **三处全修（本次落地）** | **0.963860** | **1** | 5 | 1 |

逐像素变化仅 6 个文件：
`B__resizeDiagonal` 0.0677 → **0.9973**、`C__6ab88ea05959193c` 0.2432 → **0.9797**、
`D__altText_spinner` 0.4549 → **0.9990**、`A__RKKs0ph6EDRxMJ8C` +0.0045、
`A__nDhIaDNbcrOiCCIH` +0.0028，以及下述 1 处退化。

#### 已知退化（用户已确认接受）

`C__bf975c7524dd5418` 0.9297 → **0.9073（−0.0223）**。该文件是双主题图标：
顶层 `.dark{display:none}` + `@media(prefers-color-scheme:dark){.dark{display:inline}}`。
A/B 显示去掉 `<style>` 后 gosvg 画 9199 px，与**基线**逐像素一致 —— 说明旧的 CSS
解析 bug 误打误撞把 @media 里的 `display:inline` 也应用了，抵消了 `display:none`，
恰好与「忽略 CSS、两张图都画」的 cairosvg 基准对齐。修正 CSS 语义后 gosvg
正确只画 light 主题（8976 px），少 223 px，于是相对该基准退化。
这是**修正语义带来的偏差，不是新缺陷**；该文件 IoU 仍 0.9073，三处修复的净收益
（低分 4 → 1、平均 +0.002589）明显更大，故选择在文档中记录而非回退 CSS 修复。

#### 现状

三处源码修复 + 3 个回归测试 + 本文档。**未打 tag、未发 release**，发版仍需单独确认。


### 方法论（再次校准）

1. **抽样会骗人**：117 抽样里 D 批只占 30 个、低分被当作个例；全量 848 才暴露 196 个。
2. **「成功率/像素数」不能代替几何对比**：`resizeHorizontal` 在 `cmd/corpus` 里
   100% 渲染出内容，但 IoU = 0（画在了完全不同的位置）。
3. **bbox 对比比 RGB 差可靠**：位置错时 RGB 差会假性变小。
4. 假设（even-odd）必须做**全量交叉验证**再动手，否则会去改不该改的 fork 核心。
5. **「ALL GREEN + golden 通过」不等于「修复生效」**：无退化验证与生效验证是两件事。改动渲染代码后，除 `verify.sh` 外必须做修复前后**逐文件 IoU / 像素差异**比对，确认目标文件真的动了、且没有误伤；本次正是靠这一步拦下了一个看似完整、实则零效果的 PR。
6. **聚合指标不能用来给低分定性**：bbox 与不透明像素数都近似，只能排除位置 / 尺寸错误，排除不了「内部被挖空」（反例 `B__issue-exclamation-filled` IoU 0.3497）。判断根因要做**单属性消融 A/B**（本轮 4 组实验每次只改 `fill-rule` 或 `clip-rule` 之一），并先算**富集比**再下「个例」结论：11/15 低分命中双 evenodd 声明 vs 全量 61/848，富集 ~27 倍。
7. **「对齐基准」不等于「正确」**：`C__bf975c7524dd5418` 的旧 CSS bug 恰好与 cairosvg 的「忽略 CSS」行为对齐，修好后反而退化 0.0223。改动渲染代码前要分清「修 bug」与「刷指标」，并在全量对账里把这类退化单独列出、单独决策（本轮由用户在 A / B 两方案中明确选择 A）。
