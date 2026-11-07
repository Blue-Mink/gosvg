# 安全策略

## 报告

本项目托管在内网 Gitea（`https://github.com/Blue-Mink/gosvg`）。发现问题时请：
1. 优先在仓库提 **私有** issue，或直接联系维护者 `bluemink`；
2. 涉及凭据/未授权访问的问题**不要**公开贴复现细节。

## 威胁模型

gosvg 是**库**，接收不可信 SVG 字节并渲染成位图。预期调用方在服务端或 CLI 使用。

已做的防护：
- `encoding/xml` 默认不解析外部实体（无 XXE）；未处理 `DOCTYPE` / `CDATA`。
- `parseFloat` 拒绝 `NaN` / `±Inf`（`style.go`）。
- `parseLength` 只接受已知单位后缀，未知后缀（如 `12abc`）判非法。
- `<use>` / `marker` / `gradient href` / `elementBounds` 有循环与深度上限。
- 5 个 fuzz 目标（不变量：任意输入不 panic），CI 跑 30 秒冒烟。

**仍需注意**（见 `GOSVG_GAP_TODO.md` 档 2，未实现）：
- `Render(data, size)` 目前**没有尺寸上限**：超大 `size` 会触发巨量位图分配（OOM/DoS）。
  服务端接入前请自行限制 `width` / `height`。
- 嵌套 `mask` / `clipPath` 各分配一份全尺寸 NRGBA，多层叠加为乘法级内存。
- `ImageDecoder` 由调用方注入；若回调抓取远程 URL，库侧无超时/大小限制，请在回调里自行约束。
- `ImageDecoder` 是包级变量，**并发写不安全**；应在渲染前设置一次，之后只读。
- `tspan` 嵌套、`g` 深嵌套、CSS 规则链、mask↔clip 互引暂无深度上限。
- 带 `encoding="ISO-8859-1"` 等非 UTF-8 声明的 SVG 会被 `encoding/xml` 拒绝。

## 已知边界（非漏洞）

超出子集的 SVG 特性会**静默跳过**（清单见 `GOSVG_TODO.md`「明确不支持」），
不返回错误、目前也没有运行时诊断通道。调用方若需要知道「哪块被跳过了」，
见 `GOSVG_GAP_TODO.md` 档 2 的 Diagnostics 计划。
