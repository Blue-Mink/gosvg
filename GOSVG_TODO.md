# GOSVG — 纯 Go 轻量 SVG 子集渲染库（增强 Go 生态）

目标：一个可开源、可维护、定位清晰的纯 Go SVG 子集渲染器，运行时零 cgo 零外部依赖，
覆盖 circle-flags 国旗(mask)、Lucide 图标(path+stroke)、年代唱片(text) 这类实际场景。

## 支持范围（明确子集，不做完整 SVG）
元素：svg / g / defs / use / circle / ellipse / rect / line / path / polygon / polyline / text / mask
属性：viewBox / transform / fill / fill-rule / fill-opacity / stroke / stroke-width /
      stroke-linecap / stroke-linejoin / stroke-opacity / opacity / font-size /
      text-anchor / dominant-baseline / id / mask(url)

## 底层
- golang.org/x/image/vector : path fill/stroke 光栅化（AA）
- golang.org/x/image/font + gofont/goregular + opentype : text 渲染
- encoding/xml : SVG 解析

## 进度
- [x] 项目骨架 + go.mod + 依赖（rasterx + x/image v0.47）
- [x] XML 解析 → 元素树（含 mask/gradient 索引，支持 mask 直接放 svg 下）
- [x] path `d` 解析 → vector path（全指令 M/L/H/V/C/Q/S/T/A/Z）
- [x] 基本形状渲染（circle/ellipse/rect/line/polygon/polyline）
- [x] fill/stroke/opacity 样式 + **样式继承**（svg/g → 子元素）
- [x] transform + viewBox（xMidYMid meet）
- [x] mask 合成（luminance mask，alpha 加权）
- [x] text 渲染（goregular + opentype，anchor/baseline）
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
- clip-path / stroke-dasharray / tspan / class 选择器 / <style> / preserveAspectRatio 非默认值 / mask-type=alpha
