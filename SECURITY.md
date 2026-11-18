# 安全策略

## 报告

本项目托管在 GitHub（`https://github.com/Blue-Mink/gosvg`，Private 仓库）。发现问题时请：
1. 优先在仓库提 **私有** issue，或联系维护者 `Blue-Mink`；
2. 涉及凭据/未授权访问的问题**不要**公开贴复现细节。

## 威胁模型

gosvg 是**库**，接收不可信 SVG 字节并渲染成位图。预期调用方在服务端或 CLI 使用。

已做的防护：
- `encoding/xml` 默认不解析外部实体（无 XXE）；未处理 `DOCTYPE` / `CDATA`。
- `parseFloat` 拒绝 `NaN` / `±Inf`（`style.go`）。
- `parseLength` 只接受已知单位后缀，未知后缀（如 `12abc`）判非法。
- `<use>` / `marker` / `gradient href` / `elementBounds` 有循环与深度上限。
- 6 个 fuzz 目标（不变量：任意输入不 panic），`verify.sh` 全部覆盖：
  `FuzzParse` 30s / `FuzzRender` 20s / 其余 4 个各 5s。

**已实现的输入边界**：
- `Render(data, size)` 有尺寸上限 `DefaultMaxDimension = 8192`，超限返回
  `UnsupportedError`；可通过 `RenderOptions.MaxDimension` 调整（调大请自行评估内存）。
- `mask` / `clipPath` 离屏渲染有 `maxNestingDepth = 32` 上限；超限时跳过该层
  mask/clip 并记入 `Document.Diagnostics()`。

**仍需注意**（见 `GOSVG_GAP_TODO.md`）：
- `ImageDecoder` 由调用方注入；若回调抓取远程 URL，库侧无超时/大小限制，请在回调里自行约束。
- `ImageDecoder` 是包级变量，**并发写不安全**；应在渲染前设置一次，之后只读。
  更推荐用 `RenderOptions.ImageDecoder` 逐渲染注入。`DefaultImageDecoder()` **只处理
  `data:` URI**（v0.2.4 起），不读本地文件、不抓取远程 URL；若自行实现远程解码器，
  请在回调内加超时与大小限制。
- XML 嵌套深度受 `MaxXMLDepth = 256` 限制（含 `tspan` / `g` 嵌套）；SVG 字节数受
  `MaxSVGSize`（8 MiB）、元素总数受 `MaxElementCount`（10 万）限制，超限返回对应 `Err*`。
  mask↔clip 离屏嵌套受 `maxNestingDepth = 32` 限制。
  `<style>` 的 CSS 解析也有规模上限：`MaxCSSRules = 10000`、`MaxCSSSelectorLen = 512`、
  `MaxCSSDeclLen = 4096`；超限内容被丢弃（不是让整个渲染失败），并记入 `Diagnostics()`。
- XML 编码声明：UTF-8 / ASCII 系透传、ISO-8859-1 逐字节展开，其余编码返回带编码名的
  明确错误（不再是 `encoding/xml` 那句 `CharsetReader is nil`）。替换包级 `CharsetReader`
  可扩展；置 nil 可恢复 `encoding/xml` 的默认严格行为。
- `<image>` 只有 `data:` URI 会被内置解码（`decodeDataURIOnly`，`image.go:33`）；
  **文件路径与远程 URL 必须显式注入解码器**，库不会自行读文件或发起网络请求
  （有测试守住该边界）。其余 href 静默跳过并记入 `Diagnostics()`。
- **本地文件路径是显式选择**：`DefaultImageDecoder()` 自 v0.2.4 起只解 `data:` URI。
  需要读本地文件时改用 `LocalFileImageDecoder(allowedRoots...)` —— 传入白名单根目录
  后，路径会先解析 `..` 与符号链接再校验是否越界，越界返回错误、不打开文件；远程
  URL 一律拒绝。不传 `allowedRoots` 则允许任意本地路径（等价于 v0.2.4 之前的行为），
  **仅供完全可信输入使用**。处理不可信 SVG 时应只用 `DefaultImageDecoder()`，
  或给 `LocalFileImageDecoder` 传受限根目录；本次收紧由
  `imagedecode_security_test.go` 的回归测试守住。

## 已知边界（非漏洞）

超出子集的 SVG 特性会**静默跳过**（清单见 `GOSVG_TODO.md`「明确不支持」），
不返回错误。调用方若需要知道「哪块被跳过了」，用
`Document.Diagnostics()`（累计）或 `Document.LastRenderDiagnostics()`（最近一次渲染），
实现见 `gosvg.go` / `render.go` / `image.go`；诊断并非穷尽所有静默分支。
