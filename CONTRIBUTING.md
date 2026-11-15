# 贡献指南

## 环境

- Go **1.27.3**（`go.mod`）；仓库为纯 Go、零 cgo。
- 内网环境若 `proxy.golang.org` 不通，用 `GOPROXY=https://goproxy.cn,direct`。

## 提交前必跑

```sh
gofmt -l .                          # 必须无输出
go vet ./...
go test ./...                       # 含 golden 哈希回归
go test -race ./...
GOARCH=arm64 GOOS=linux go build ./...
```

CI（`.gitea/workflows/ci.yml` 与 `.github/workflows/ci.yml` **内容保持一致**）只调
`sh ./scripts/verify.sh`（含 6 个 fuzz 目标、`go mod tidy -diff`、wasm 构建），与本地同源。
改了任一 CI 文件请同步改另一个。注意 **Gitea Actions runner 数为 0**、GitHub 出站不通，
CI 目前未在服务端真正执行，提交前请本地实跑 `verify.sh`。

> 量化现状（2026-11-15 查 Gitea API `/actions/runs`）：累计 **113 次 run，
> `conclusion=success` 0 次**（106 cancelled + 7 queued，queued 为 #107–#113，
> 由 v0.3.0 发布相关 push/PR 触发），`actions/runners` 返回 `total_count=0`。
> 即：**没有任何一次 PR / push 被自动验证过**。注册 act_runner 需动 `.15` 上的
> 服务、且当前 GitHub 出站不通，须用户明确授权后才做；清理 queued run 同样会
> 写 `.15` 的 Gitea，不在本地改动范围内。

## 提交风格

Conventional Commits + 中文描述，沿用现有提交风格：`feat:` / `fix:` / `docs:` /
`chore:` / `test:`。一个提交一个主题，不要把文档、CI、代码混成一个巨型提交。

## 改了渲染逻辑

- 若**有意**改变渲染输出，重新生成基线：`go test -run TestGoldenRegression -update .`，
  并把 `testdata/golden.sha256` 的变更连同理由写进提交信息。
  ⚠️ 该 flag **跳过全部比对**并直接覆盖基线；常规开发 / CI 绝不传它。新增
  `testdata/*.svg` 会被自动枚举进基线，无需手改列表。
- 与 cairosvg 的对账用入仓工具 `cmd/iou`（v0.2.4 起随仓库提供，取代仓库外一次性
  脚本；用法见 `README.md`）；`alphaAAD` 口径仍未实现。数字更新请同步
  `CHANGELOG.md` / `CAIROSVG_IOU.md`，并跑 `sh ./scripts/verify.sh`（含文档一致性门禁
  `scripts/check_docs.sh`：既校验 `.md` 的版本/数字/路径不变量，也禁止在 `.go` /
  `.sh` / `.py` 注释里写死固定样例数）。发版前建议再跑
  `python3 scripts/audit_docs.py` 做深度审计：它列出全仓相对时间词、数字声明、
  版本号分布、路径引用，以及 F 段「源码注释里的数字/时间声明」，供人工逐条确认
  是否有过期表述（只报告、不阻塞，历史快照按纪律保留原文并就地加注，不要删改
  当时结论）。**别在注释里写死样例数** —— 数量以 `testdata/*.svg` 自动枚举为准。

## 改了 fork 代码

`internal/rasterx`（MIT，srwiley/rasterx）与 `internal/vector`（BSD-3-Clause，x/image/vector）
是 fork，改动请同步更新同目录 `UPSTREAM.md`，并保留 `LICENSE`。

## 安全

- 提交前确认新增内容不含真实口令、token、内网凭据。
- 已知输入边界与报告方式见 `SECURITY.md`。
