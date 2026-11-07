# GOSVG — 纯 Go 轻量 SVG 子集渲染库（增强 Go 生态）

目标：一个可开源、可维护、定位清晰的纯 Go SVG 子集渲染器，运行时零 cgo 零外部依赖，
覆盖 circle-flags 国旗(mask)、Lucide 图标(path+stroke)、年代唱片(text) 这类实际场景。

## 支持范围（明确子集，不做完整 SVG）
元素：svg / g / defs / use / circle / ellipse / rect / line / path / polygon / polyline /
      text / tspan / mask / clipPath / linearGradient / radialGradient / stop
属性：viewBox / transform / fill / fill-rule / fill-opacity / stroke / stroke-width /
      stroke-linecap / stroke-linejoin / stroke-miterlimit / stroke-dasharray / stroke-dashoffset /
      stroke-opacity / opacity / gradientUnits / gradientTransform / spreadMethod / offset /
      stop-color / stop-opacity / mask / maskUnits / maskContentUnits / clip-path / clipPathUnits /
      font-size / font-family / font-weight / font-style / text-anchor / dominant-baseline /
      x / y / dx / dy / color / currentColor / id / 内联 style
颜色：hex(3/4/6/8) / rgb()/rgba() / hsl()/hsla() / 命名颜色 / none / currentColor

## 底层
- golang.org/x/image/vector : path fill/stroke 光栅化（AA，已扩展 even-odd winding）
- golang.org/x/image/font + gofont/goregular + opentype : text 渲染
- encoding/xml : SVG 解析

## 进度
- [x] 项目骨架 + go.mod + 依赖（rasterx + x/image v0.47）
- [x] XML 解析 → 元素树（含 mask/gradient/clipPath 索引，支持 defs 直挂 svg 下）
- [x] path `d` 解析 → vector path（全指令 M/L/H/V/C/Q/S/T/A/Z）
- [x] 基本形状渲染（circle/ellipse/rect/line/polygon/polyline）
- [x] fill/stroke/opacity 样式 + **样式继承**（svg/g → 子元素）
- [x] transform + viewBox（xMidYMid meet）
- [x] mask 合成（luminance mask，alpha 加权；maskUnits/maskContentUnits/transform）
- [x] clip-path（形状 alpha 裁剪；clipPathUnits/transform/区域/多子元素 union）
- [x] fill-rule="evenodd" 真实现（vector 光栅器补 even-odd winding）
- [x] gradient（linear/radial + units/transform/spread/stop/href 继承，fill 与 stroke 均支持）
- [x] text 渲染（goregular + opentype，anchor/baseline）
- [x] tspan（text 内嵌片段，嵌套/定位/样式覆盖）
- [x] defs/use 引用
- [x] 与 cairosvg 对账（IoU + alphaAAD）
- [x] **B 方案：自研胶囊体并集 stroke（cairo 级质量，2026-11-07 完成）**
  - stroke 弃用 rasterx Stroker 的「整体偏移轮廓 + non-zero」（内偏移退化时
    自交出错，flatten 精度实验证明与 tol 无关），改为每条 flatten 线段独立
    生成「偏移矩形 + 两端半圆」的胶囊体，全部 non-zero 累积进一个
    x/image/vector Rasterizer 填充——round join 由相邻端半圆自然重叠实现，
    内偏移退化天然免疫。rasterx fork 进 internal/rasterx 继续负责 fill。
  - 途中修掉 5 个 bug：半圆绕向（cap 覆盖反侧）、多子路径连线（子路径必须
    独立不连线）、M-only 子路径不渲染、addCircle 与 addCapsule winding 方向
    相反（同框抵消镂空）、vector 通用路径 alpha=0 幽灵 RGB 清理。
  - 对账（cairosvg 512px IoU）：path1（小半径 arc，旧 0.9443）→ **0.9911**；
    rock 0.9939 / country 0.9968 / blues 0.9961 / jazz 0.9944 / fill 0.9998；
    多子路径 0.9997+、零长段 0.9879、circle_path 0.9950；47ms/图。
  - 扇谷效应边界：凸曲线外侧相邻半圆交点内缩 ≈ R·tol/u（tol=0.01 时
    <0.05px 亚像素级），再降 tol 收益递减（AA 算法差异成主导）。

## 与 cairosvg 对账结果（512×512）
| SVG | IoU | alphaAAD | 备注 |
|---|---|---|---|
| cn 国旗（fill+mask） | 0.9998 | 0.1 | 完美 |
| country | 0.9950 | 1.3 | |
| blues | 0.9940 | 1.4 | |
| rock guitar | 0.9919 | 1.9 | path1（小半径 arc+粗 stroke）0.9443 |
| jazz | 0.9923 | 1.8 | |

## 修掉的关键 bug
1. stroke 线宽没随 viewBox 缩放（rasterx SetStroke 不走 MatrixAdder，需手动乘 scale）——曾经线条只有 2px
2. path tokenize：`.9.4` 紧凑小数写法被当一个 token → 参数错位
3. luminance mask 没乘 alpha（白 RGB+alpha0 被当亮度 255）
4. 离屏渲染 sub.style 零值 → fillOpacity=0 全透明
5. mask 元素直接在 svg 下没被索引（circle-flags 的写法）
6. renderMasked 无限递归

## 已知局限（rasterx stroke 底层）
- 小半径曲线（arc 半径≈stroke 半宽）的内偏移退化：path1（吉他头）IoU 0.94，
  孔边缘比 cairo 略小/略尖。超采样无效（非 AA 问题，是 stroke 轮廓几何差异）。
- fill 类场景与 cairosvg 几乎无差别（0.9998）。

## 待办
- [ ] 开源准备：README / LICENSE / 示例
- [ ] go vet：transform.go Matrix2D unkeyed fields 警告
- [ ] （可选）stroke 轮廓算法自研，解决小半径退化

### v3 锯齿优化（2026-11-07 用户反馈"边缘还有锯齿"后）
- 填充器 vector→rasterx.Filler（与 fill 同一套扫描器，AA 与 fill 同水平；
  两者与 cairosvg 对齐度差异实测 fill圆 0.9986 vs 0.9952）
- 两个截断 bug（都改成四舍五入）：
  1. internal/rasterx fill.go：CubeTo/QuadTo 的 lineF 回调 fixed.Int26_6(ex)
     直接截断 float32，每个 flatten 顶点偏移 ±0.5 fixed，产生方向性噪声
     （表现为红/绿差异像素随 flatten 段数翻转 459↔52）
  2. path.go fpt：坐标量化截断，所有 path 几何向 -x/-y 系统偏移
- tol 语义修复：flatten 容差从用户单位改为像素（原 0.01 用户单位=0.21px）
- IoU 终值（cairosvg 512px）：path1 0.9987 / rock 0.9985 / circle 0.9978 /
  country 0.9968 / blues 0.9962 / jazz 0.9959 / cn(fill) 0.9998；49ms/图
- 结论：剩余差异 = rasterx AA 与 cairosvg 的亚像素舍入差（红绿平衡各~50px），
  4x 边缘放大肉眼不可辨；已到该 AA 基线（fill 同为 0.9986）

## 审计修复（2026-11-07，全面代码审查后）

### 修复的渲染 bug
1. text 的 opacity 被重复乘 ctx.opacity（`opacity*fillOpacity*ctx.opacity` → 0.5×0.5=0.25 而非 0.5）
2. text 的 fill/fill-opacity 不继承父级样式（`<g fill="red"><text>…</text></g>` 失效）
3. 基本形状/text 元素**自身**的 transform 属性被忽略（renderShape 不处理 transform）
4. 基本形状/text 元素**自身**的 opacity 属性被忽略（mergeStyle 不处理 opacity）
5. path 科学计数法：`e`/`E` 被误当指令字母切分；`1e-5` 被拆成 `1e`+`-5`
6. width/height/font-size 带单位（`512px`、`12pt`）解析失败

### 清理
- 死代码：`parseLength`（重复 TrimSuffix）、`capFunc`/`joinMode`（stroke 改精确轮廓后废弃）
- `renderContext.opacity` 字段（重构后无读取者）
- `effectiveStyle` 残留注释

### 新增能力
- 内联 `style="fill:red;stroke:blue"`（优先级高于 presentation 属性）
- `parseLength` 正确处理 px/pt/em/% 单位

### 回归测试
- `fix_test.go`：text opacity/fill 继承、shape transform/opacity、科学计数法、parseLength、内联 style
  —— 旧字节 6 项全 FAIL → 新字节全 PASS（证明用例咬得住）

### 未实现（已知边界，见 gosvg.go 包注释）
- gradient（grads 已索引但未渲染，fill="url(#id)" 降级为不填充）
- clip-path / tspan / class 选择器 / <style> / preserveAspectRatio 非默认值 / mask-type=alpha

## stroke-dasharray / stroke-dashoffset（2026-11-07 完成）
- 实现：`dash.go`（parseDashArray / normalizeDashes / dashSegments / dashRanges /
  polylineLength / polylinePointAt / extractPolyline），参考 tdewolff/canvas `Path.Dash`。
- 语义：奇数长度 dasharray 自动复制成偶数（SVG 规范）；dash 段在 flatten 后的折线上按弧长
  切分，每段独立走 strokeExactPath（开放、沿用原 linecap/linejoin）；闭合子路径展开成环，
  跨起点的 dash（offset 使 dash 越过路径起点）把首末两段合并。
- 对账（cairosvg 512px IoU）：水平线/斜线/矩形/奇数数组/dashoffset **1.0000**，
  round-cap **0.9998**，曲线 **0.9957**（flatten 正常差异），闭合多边形 **0.9765**
  （见下方"既有斜线 stroke AA 差异"，非 dash 引入）。
- 单元测试 `dash_test.go` 14 项全绿。

## 既有问题（非本轮引入）：45° 斜线 stroke AA 比 cairosvg 略细（已排查定案）

### 现象
- 45° 斜线（polygon 菱形、无 dash）IoU 0.9698，红像素 860 均匀散布在 4 条边（各 215），
  join 类型（miter/round/bevel）都同值 → 与 join 无关。
- 单条 line `(50,5)→(95,50)`（斜率+1，中点半像素）0.9678；`(50,10)→(90,50)`（中点整数）满分。

### 根源（2026-11-07 排查定案）
- **vector.Rasterizer（internal/vector = x/image/vector 的 libart 扫描线 AA）对
  "恰好穿过像素角的 45° 斜线"（c 整数相位）有 ±1px 覆盖歧义**。fill 和 stroke 都走
  这个 rasterizer，所以**两者都受影响**（实证：fill 26.57° 斜线 offset 平行四边形
  也是 0.9897，fill 45° 整数坐标菱形才满分）。
- stroke 比 fill 更容易触发，是因为 stroke 的 offset 坐标含无理数（√2），fpt 四舍五入
  后 c 变成"接近整数但非整数"的相位，恰好落在 AA 的歧义区。
- 试验过"改 stroke 坐标量化为 fpt(用户)+TFixed(截断)"（对齐 fill 的 MatrixAdder）：
  菱形 A 0.9698→0.9988，但 **cap_butt/cap_round（26.57°）从 1.0 掉到 0.9897**，
  且 fill 用同路径对 26.57° 也 0.9897 → TFixed 截断是有偏的，引入更多回归，已回退。
- cairo 用超采样，任何相位都精确。

### 影响与决定
- 实际图标（Lucide rock/country/blues/jazz/lucide）对账 0.9946~0.9985，45° 斜线相位
  在真实图标里被曲线与多角度混合，**无显著影响**。
- 修复只有超采样一条路（4x 内存/时间），对 1px 偏差收益有限 → **接受为 vector AA 的
  已知边界**，不修。保持 fpt round（对称、盲点用例几乎全满分）优于 TFixed 截断。


## 参考 tdewolff/canvas 的调研结论（2026-11-07）

用户给的参考（gitcode 镜像 + github 原仓都是同一个 `tdewolff/canvas`）：

- **定位相反**：tdewolff/canvas 是"全家桶"式多后端矢量库（PDF/SVG/EPS/OpenGL/Gio/Fyne/HTMLCanvas/WASM + 文本整形 HarfBuzz/FriBidi），go.mod 直接 require fyne、gioui、go-gl、freetype、oksvg、rasterx 等，**247 个 go.sum 依赖**。gosvg 的 go.mod 只有 `golang.org/x/image` 一个依赖——gosvg 的"零 cgo 零外部依赖"定位恰好是它的反面。**结论：只能当算法参考，绝不能当依赖。**
- 有趣事实：tdewolff/canvas 自己都间接依赖 `srwiley/rasterx`（via oksvg）——和 gosvg 用的同一个 rasterx。
- 它现在有完整 SVG 解析+渲染（svg.go 1532 行 + svg_parse_test.go），支持 linearGradient/radialGradient/marker/pattern/stroke-dasharray/clip-path/CSS `<style>`/class 选择器/tspan。

### 算法对照结论（逐项核实过）
| 领域 | tdewolff/canvas | gosvg 现状 | 结论 |
|---|---|---|---|
| arc→cubic | Maisonobe kappa（path_util.go `ellipseToCubicBeziers`） | `4/3·tan(dt/4)` | 圆已最优；椭圆 tdewolff 略准，但收益 <0.1%，不追 |
| circle flatten | 直接解析式（path_util.go `flattenEllipticArc`） | cubic + de Casteljau | 已够好 |
| stroke | 双偏移曲线 + settle（path_stroke.go `offset`/`Stroke`） | 精确轮廓（offset line + arc join） | 已 cairo 级 0.99+，不换 |
| **stroke-dasharray/dashoffset** | `Path.Dash`（path.go:1856） | **未实现**（render.go `dasher` 字段闲置） | **可参考实现** |
| gradient / clip-path / marker | 完整实现 | 未实现 | 后续按需参考 |

### 下一步建议（按价值排序）
1. **stroke-dasharray + dashoffset**（自包含、可对 cairosvg 验证，参考 `Path.Dash`）
2. **gradient（linear/radial）**（最高价值但复杂：需在光栅器里做逐像素轴向插值 + gradientUnits）
3. clip-path / marker（低频，往后放）

## 系统字体支持 font-family（2026-11-07 完成，方案 B）
- 动机：内置 Go Regular 的 0 天生是斜杠零、笔画偏细，与 cairosvg 用系统 DejaVu 渲染的
  年代唱片标签（"80"/"90"）字形对不上。
- 实现：`font.go`——`AddFontDir(dir)` 递归扫 .ttf/.otf，读 NameIDFamily/Subfamily
  按 (family, 字重, 字型) 注册；`renderText` 解析 `font-family`（逗号列表、去引号、
  CSS 通用名 sans-serif/serif/monospace → DejaVu 优先）+ `font-weight/font-style`
  择优字型；未命中回退内置 Go Regular（不调用 AddFontDir 时行为与旧版字节一致）。
- 字重判定：subfamily 名含 bold → bold，含 italic/oblique → italic（DejaVu 的
  regular subfamily 是 "Book"，正确归为 regular）；font-weight 数值 ≥600 算粗体。
- 回归 `font_test.go` 7 项全绿：字重/字型判定表、逗号列表拆分、未注册回退 nil、
  双指针字型选择与逐级回退、通用名映射、**端到端**（未注册 = Go Regular 字节一致；
  注册 DejaVu 后渲染字节改变 = 真换字体）。
- 对比验证（80s 标签 SVG）：cairosvg 与 gosvg(DejaVu) 的 "80" 均为 DejaVu Sans Bold
  无斜杠零，字形一致；Go Regular 回退列仍是斜杠零细体。
- 边界（写进包注释）：引入"渲染结果取决于系统字体"的边界——不调用 AddFontDir 则
  保持零依赖不变；.ttc 集合字体暂不支持。

## gradient 实现（2026-11-07 完成，fill 支持 linear/radial）
- 新增 `gradient.go`：`resolveGradient` / `buildGradient` / `fillGradient` / `pathBounds`。
- 复用 fork 的 `internal/rasterx` 已有 `Gradient` 结构 + `GetColorFunctionUS`（返回 ColorFunc，
  ScannerGV.SetColor 原生支持 ColorFunc）——底层光栅化能力本就齐备，只需补解析层与 fill 接线。
- 支持：linearGradient / radialGradient、gradientUnits（userSpaceOnUse / objectBoundingBox）、
  gradientTransform、spreadMethod（pad/reflect/repeat）、stop（offset/stop-color/stop-opacity，
  offset 支持百分比）、href/xlink:href 继承（stops 与通用属性继承，几何由当前元素决定）、
  fill-rule 组合（non-zero）。
- objectBoundingBox 的 bbox 用 `pathBounds`（flattenPath 后取极值，device 空间）。
- 测试 `gradient_test.go` 6 项：linear(obb)/radial(uso)/href 继承/三 stop 插值/stop-opacity。
- **顺带发现既有问题**：`fill-rule="evenodd"` 从未生效——`ScannerGV.SetWinding` 是 no-op，
  `internal/vector`（fork x/image/vector）光栅器仅支持 non-zero（源码 TODO 注释）。已写进
  包注释与 README 的"明确不支持"，暂不实现（需在 vector 光栅器里补 even-odd 扫描）。

## stroke-miterlimit（2026-11-07 完成）
- 新增 `shapeStyle.miterLimit`（默认 4），`mergeStyle` 解析 `stroke-miterlimit`。
- 语义修正：旧 `emitJoin` 硬编码 `miterLimit=4` 且判断 `d <= miterLimit*u`（实际等价
  miterlimit=2，偏保守）；改为 SVG 规范的 `d <= 2*miterLimit*u`（miterLength/strokeWidth）。
- 测试 `fix_test.go` TestStrokeMiterLimit（90° 角 0.5→bevel / 1→miter）+ TestMergeStyleMiterLimit。

## 修复的实证 bug（2026-11-07）
- stroke-width 带单位（"4px"）解析失败回退 1 → 改 parseLengthOr。
- text fill="none"/"url(#)" 渲染成黑色 → 解析失败 return（与 renderShape fill 对齐）。
- use 循环引用栈溢出 → renderContext.useDepth 深度上限（maxUseDepth=100）。

## mask 的 maskUnits/maskContentUnits/transform（2026-11-07 完成）
- 重写 `mask.go` 的 `renderMaskImage`：支持 maskContentUnits（userSpaceOnUse 默认 /
  objectBoundingBox 归一化映射到 bbox）、mask 自身 transform、maskUnits 的
  x/y/width/height 区域裁剪（objectBoundingBox / userSpaceOnUse）。
- 新增 `elementBounds`（遍历元素树算 device 空间几何 bbox，支持 g/use/形状，depth 防循环）
  与 `applyMaskRegion`（区域外亮度归零）。
- 之前 mask 一律当 userSpaceOnUse + 无区域裁剪 + 忽略 transform；circle-flags 恰好是
  默认值下等价，才过对账。现在完整支持。
- 测试 `mask_test.go` 5 项：maskContentUnits obb（全/部分覆盖）、mask transform、
  maskUnits region（userSpaceOnUse + objectBoundingBox）。

## hsl()/现代颜色（2026-11-07 完成）
- `style.go` 重构 `parseColor`：新增 `parseColorFunc` 统一解析 rgb()/rgba()/hsl()/hsla()。
- 支持 hsl/hsla（h 度/deg、s/l 百分比或 0..1）、rgb 空格分隔语法（`rgb(255 0 0)`）、
  斜杠 alpha（`rgb(255 0 0 / 0.5)`），顺带保留 rgba 逗号第 4 分量老语法。
- 新增 `hslToRGB`/`hueToRGB`/`parseHue`/`parsePercent`/`parseAlpha`。
- 测试 `color_test.go` 4 项：hsl 六基色+空格/deg、hsla alpha+斜杠、rgb 空格+斜杠、
  渲染链路 hsl 填充。

## fill-rule="evenodd" 真实现（2026-11-07 完成）
- 根因：`ScannerGV.SetWinding` 是 no-op，`internal/vector`（fork x/image/vector）光栅器
  仅 non-zero（accumulate 取 |winding| 绝对值）。
- 修法：`vector.Rasterizer` 加 `UseEvenOdd` 字段；`accumulateMask` 按 even-odd 分支（禁用
  SIMD，走新增的 `floatingAccumulateMaskEvenOdd` / `fixedAccumulateMaskEvenOdd`）；`Draw`
  在 even-odd 时跳过 *image.Uniform fast path（否则 SIMD rasterize 绕过 accumulateMask）；
  `ScannerGV.SetWinding` 真正写 `UseEvenOdd = !useNonZeroWinding`。
- even-odd 数学：winding 数 = ceil(|acc|/unit)，奇数→填充（alpha=部分覆盖），偶数→透明。
  unit = fxOne²（fixed）/ 1.0（floating）。
- 测试 `evenodd_test.go` 4 项：同心矩形镂空、evenodd vs nonzero、winding=3 奇偶、evenodd+gradient。

## gradient stroke（2026-11-07 完成）
- stroke 也支持 url(#id) 渐变：`stroke` 拆成 `stroke`(纯色)/`strokeGradient`(渐变)/
  `strokeColor`(通用，clr 可为 ColorFunc)；`strokeCapsules` 的 c 参数改 interface{}。
- objectBoundingBox 用 pathBounds 算路径几何 bbox（不含 stroke 宽度）。
- 测试 gradient_test.go +2：userSpaceOnUse 水平线、objectBoundingBox rect 描边。

## clip-path（2026-11-07 完成）
- 新增 `clip.go`：`renderClipped` / `renderClipMask` / `clipPathRef`。
- 语义：clip 用形状覆盖（alpha）裁剪，不做亮度换算（区别于 mask 的 luminance）。
- 支持 clipPathUnits（userSpaceOnUse 默认 / objectBoundingBox）、clipPath 自身 transform、
  x/y/width/height 区域（复用 applyMaskRegion）、多子元素并集。
- renderNode 先 clip 后 mask；renderNodeBare 跳过 clippath 元素。
- 测试 `clip_test.go` 3 项：基本裁剪、clipPathUnits obb、多子元素 union。

## tspan（2026-11-07 完成）
- 重构 `text.go`：`renderText` 拆成 `renderText`/`renderTextSpans`/`drawTextSpan`，
  沿 x 游标排布「直接文本 + tspan 子元素」，支持嵌套。
- tspan 支持样式继承覆盖（fill/font-size/font-family/font-weight/font-style/
  text-anchor/dominant-baseline）、定位（x/y 绝对、dx/dy 相对）、绝对 x 重置游标。
- drawTextSpan 返回用户空间宽度供游标前进。
- 测试 `tspan_test.go` 5 项：仅 tspan、直接文本+tspan 共存、fill 继承、嵌套、x 偏移。
