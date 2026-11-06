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
