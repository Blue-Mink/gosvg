# GoSvg 使用文档

> 纯 Go 实现的通用 SVG 渲染命令行工具集 · v0.4.1 · 本文所有示例均为**真实运行结果**。
>
> 开源上游：[github.com/Blue-Mink/gosvg](https://github.com/Blue-Mink/gosvg)

> [!NOTE]
> 本文档由飞牛 `GoSvg` 应用「打开」入口实时从 GitHub 仓库加载。修改本文档只需更新仓库中的 `USAGE.md`，无需重新打包 FPK。

## 简介与准备

GoSvg 是用 Go 编写的 SVG 渲染库与命令行工具集，追求 SVG 标准兼容与渲染一致性。飞牛 FPK 包把 4 个核心工具打包成 NAS 上的 CLI 程序，**没有 Web 操作界面**——本文档只是说明，实际操作通过 SSH 登录后在命令行完成。

工具统一位于 `/vol1/@appcenter/gosvg/bin/`。示例前先设一个变量简化输入：

```sh
# 设为工具目录（后面 $B 即代表它）
B=/vol1/@appcenter/gosvg/bin

# 准备一个示例 SVG（demo.svg）
cat > demo.svg <<'EOF'
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 120">
  <rect width="200" height="120" fill="#eef3ff"/>
  <circle cx="60" cy="60" r="40" fill="#2f9bff"/>
  <rect x="110" y="30" width="60" height="60" rx="10" fill="#ff6b8a"/>
  <text x="100" y="112" font-size="16" text-anchor="middle" fill="#15357a">GoSvg</text>
</svg>
EOF
```

下文的示例都基于这个 `demo.svg`，你可以直接复制照做。参数以各工具的 `-h` 输出为准。

## 1. gosvg-render — 单文件渲染

把一个 SVG 文件渲染成 PNG 或 JPEG 位图。输入/输出是**位置参数**（输入在前、输出在后），也可以用 `-` 走标准输入输出。

### 参数表

| 参数 | 说明 |
|:--|:--|
| `-size N` | 宽高相同，默认 512；被 `-W`/`-H` 覆盖 |
| `-W N` / `-H N` | 分别指定输出宽、高（0 = 用 -size） |
| `-bg #RGB\|#RRGGBB` | 不透明底色，默认透明；JPEG 透明区默认合成白色 |
| `-format png\|jpeg` | 输出格式；省略则按扩展名推断（.jpg/.jpeg→jpeg，其余→png） |
| `-quality N` | JPEG 质量 1–100，默认 90 |
| `-strict` | 渲染期出现静默跳过（未识别元素、缺失引用等）则退出码 3 |
| `-` | 输入为 `-` 时从 stdin 读；省略输出路径时写到 stdout |

### 实例 A：基础渲染（256×256 PNG）

```sh
$ $B/gosvg-render -size 256 demo.svg out.png
# 退出码 0，生成 out.png（6023 字节，PNG 尺寸 256×256）
```

### 实例 B：指定宽高 + 白底 JPEG

```sh
$ $B/gosvg-render -W 400 -H 240 -bg '#ffffff' -format jpeg -quality 85 demo.svg out.jpg
# 退出码 0，生成 out.jpg（7300 字节）
```

### 实例 C：stdin → stdout（管道）

```sh
$ cat demo.svg | $B/gosvg-render -size 128 - > stdin.png
# 退出码 0，stdin.png 2484 字节
```

### 实例 D：严格模式

```sh
$ $B/gosvg-render -strict -size 64 demo.svg strict.png
# 正常 SVG：退出码 0；若有被跳过的元素则退出 3 并打印诊断
```

> [!TIP]
> 旧写法 `gosvg-render in.svg out.png 256` 仍兼容；但位置 size 与 `-size`/`-W`/`-H` 同时出现会报错退出 2。

## 2. gosvg-corpus — 批量语料渲染

对一个目录里的所有 SVG 批量渲染，输出统计报告。目录作为位置参数传入。

### 参数表

| 参数 | 说明 |
|:--|:--|
| `-size N` | 渲染边长（正方形），默认 64 |
| `-json` | 以 JSON 输出报告（默认是人类可读文本） |

### 实例：批量渲染 2 个 SVG

```sh
$ mkdir svgs && cp demo.svg svgs/a.svg && cp demo.svg svgs/b.svg
$ $B/gosvg-corpus -size 64 svgs
语料目录: svgs
SVG 文件数: 2   渲染尺寸: 64x64   耗时: 6ms
渲染出内容: 2 (100.0%)
渲染全透明: 0 (0.0%)
Parse 失败: 0 (0.0%)
Render 失败: 0

# 加 -json 得到结构化报告，便于脚本处理
$ $B/gosvg-corpus -size 64 -json svgs
{ "dirs":["svgs"], "size":64, "total":2, "with_content":2,
  "blank":0, "parse_failed":0, "render_failed":0, "elapsed_ms":6 }
```

## 3. gosvg-iou — 渲染一致性对账

把一个 SVG 目录渲染后，与参照图目录里的 PNG 逐文件计算 IoU（交并比）相似度，用于回归测试/渲染对账。SVG 文件名去掉扩展名后与参照 PNG 名对应（如 `a.svg` ↔ `a.png`）。

### 参数表

| 参数 | 说明 |
|:--|:--|
| `-svg 目录` | 待渲染的 SVG 目录（与 -baseline 至少给一个） |
| `-ref 目录` | 参照图目录，内含 PNG（必填） |
| `-baseline 目录` | 基线渲染目录；给定则额外算与基线的 IoU 差异 |
| `-size N` | 渲染边长，默认 256 |
| `-alpha N` | alpha 阈值；0=alpha>0 视为不透明，8=报告口径 |
| `-min F` | 只列出 IoU 低于 F 的文件（0=不限制） |
| `-out 目录` | 渲染输出目录；为空则不落盘 |
| `-json 文件` | 把完整结果写成 JSON 到指定文件 |
| `-quiet` | 只输出汇总，不逐文件打印 |

### 实例 A：正常对账（渲染图与参照一致）

```sh
$ mkdir refs && cp out.png refs/a.png && cp out.png refs/b.png
$ $B/gosvg-iou -size 256 -svg svgs -ref refs

alpha>0  size=256  compared=2  missing=0
average IoU = 1.000000
IoU < 0.5   = 0
```

### 实例 B：制造差异 + 阈值过滤（找出退化项）

```sh
# 把 refs/b.png 换成一张明显不同的图后：
$ $B/gosvg-iou -size 256 -svg svgs -ref refs -min 0.9
LOW  b                                  iou=0.0000 ref=1 got=0

alpha>0  size=256  compared=2  missing=0
average IoU = 0.500000
IoU < 0.5   = 1
```

### 实例 C：导出 JSON 报告

```sh
$ $B/gosvg-iou -size 256 -quiet -json iou.json -svg svgs -ref refs
# iou.json 含 compared / average_iou / samples[]（每个文件的 iou、像素数）
```

> [!TIP]
> 若只给 `-baseline` 不给 `-svg`，则只做两组已有图的比对；两者都省略会提示"至少需要 -svg 或 -baseline 之一"。

## 4. gosvg-verify — 项目验证门禁

这是面向**源码开发**的验证工具（普通使用者一般用不到）：在 Go 源码目录下运行完整验证门禁——单元测试、竞态检测、交叉编译、fuzz 与 golden 基线比对。

```sh
$ cd /path/to/gosvg
$ $B/gosvg-verify
# 依次执行各检查项并输出结果；需在与源码布局匹配的目录中运行
```

此工具针对开发场景，依赖完整的 Go 源码树与测试语料；打包进 NAS 仅供完整性，日常渲染用不到它。

## 常见问题

### Q：应用中心显示 non-running，工具还能用吗？

能。显示状态由轻量哨兵进程维持，仅用于满足 start/stop/status 生命周期，与 CLI 无关。只要 `bin/` 下二进制还在，SSH 调用照常可用。

### Q：支持哪些 SVG 特性？

支持常见元素（path/形状/text/渐变/mask/clipPath/marker/pattern/style 等）与 CSS 选择器级联，含 `vector-effect="non-scaling-stroke"`。完整支持清单见上游 README。明确不支持：filter、textPath、foreignObject、动画/脚本、CSS 伪类等（命中时静默跳过，可用 `-strict` 发现）。

### Q：中文文字渲染正常吗？

默认用内置 Go Regular 字体（零外部依赖），可能不含中文字形。需要中文时请在上游库中通过 `AddFontDir` 注册系统字体目录后自行构建。

---

*GoSvg · 由 Blue-Mink 构建 · 文档随 v0.4.1 打包 · 示例均为真实运行结果*

<!-- E2E_LIVE_UPDATE_MARK_1795476270 -->
