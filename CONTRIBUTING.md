# 贡献指南

## 环境

- Go **1.27.3**（`go.mod`）；仓库为纯 Go、零 cgo。
- 内网环境若 `proxy.golang.org` 不通，用 `GOPROXY=https://goproxy.cn,direct`。

## 提交前必跑

```sh
gofmt -l .                          # 必须无输出
go vet ./...
go test ./...                       # 含 golden 哈希回归
go test -race ./...                 # 需要 CGO_ENABLED=1 + 可用的 C 编译器（gcc）
GOARCH=arm64 GOOS=linux go build ./...
```

或一条命令跑齐（推荐）：`sh ./scripts/verify.sh` —— 含 `go mod verify` /
`tidy -diff` / gofmt / vet / cover / race / arm64 / wasm / bench /
6 个 fuzz 目标 / `check_docs.sh`。**在无 C 编译器的最小环境里，`verify.sh`
会跳过 `-race` 并继续跑完其余门禁，结尾如实标注
`ALL GREEN (race SKIPPED: no C compiler)`；这不是全绿，别拿它当完整验证。**

> **发版 / 合并到 main 前的硬要求**：必须在**带 gcc 的机器**上完整跑一次
> `sh ./scripts/verify.sh`，确认末行是**不带 SKIPPED 的** `ALL GREEN`
> （即 `-race` 真正实跑通过），并在发布说明/PR 里记下执行机器与结果。
> 内网 Gitea Actions 无法代劳此事（见下），本地精简容器也不行。
> 这是当前唯一尚未由自动门禁覆盖的验证项，靠流程补齐，不能靠文档角落一句话带过。
>
> ⚠️ **首次破例记录（PR #32，2026-11-15）**：硬要求立起后的第一个合并（PR #32）
> 因本机无 gcc/clang（`CGO_ENABLED=0`），验证结果为
> `ALL GREEN (race SKIPPED: no C compiler)`，**未满足**上面「不带 SKIPPED」的要求。
> 该 PR 为纯文档 / CI 配置 / 测试、零渲染代码改动、不触及并发路径，故按
> `CHANGELOG.md` `[Unreleased]` 首条如实记账；**规则不放宽**：后续凡触及渲染代码的
> 合并仍须先在带 gcc 机器跑出不带 SKIPPED 的 `ALL GREEN`。此处是记录例外，不是豁免条款。

CI（`.gitea/workflows/ci.yml` 与 `.github/workflows/ci.yml` **内容保持一致**）只调
`sh ./scripts/verify.sh`，与本地同源。改了任一 CI 文件请同步改另一个（
`check_docs.sh` 有逐字一致门禁）。

> **2026-11-15 止血**：两份 workflow 的触发方式已从
> `push`（全分支 + tags）+ `pull_request` + `workflow_dispatch`
> **收窄为仅 `workflow_dispatch` 手动触发**。原因：Gitea Actions runner 数为 0、
> GitHub 出站不通，自动触发只会持续产生永不执行的 queued run。配置保留作将来
> 恢复自动 CI 的同源模板，恢复条件与步骤写在 workflow 文件头注释里。
>
> 止血当时的量化取证（2026-11-15 查 Gitea API `/actions/runs`；按纪律保留原文，
> 当前状态见紧随其后的「状态更新」条）：累计 **113 次 run，
> `conclusion=success` 0 次**（106 cancelled + 7 queued，queued 为 #107–#113，
> 由 v0.3.0 发布相关 push/PR 触发），`actions/runners` 返回 `total_count=0`。
> 即：**没有任何一次 PR / push 被自动验证过**，且收窄后不再新增僵尸 run。
> 注册 act_runner 需动 `.15` 上的服务、且当前 GitHub 出站不通，须用户明确授权
> 后才做；清理已存在的 7 个 queued run 同样会写 `.15` 的 Gitea，不在本地改动范围内。
>
> ⚠️ **状态更新（PR #32 合并后，2026-11-15）**：上面这组「106 cancelled +
> 7 queued」是止血当时的取证快照，按纪律保留原文。现 7 个 queued run 已逐个取消，
> **113 run 全 `cancelled`、`queued=0`**；两个 commit 已推并经 PR #32 合并，
> 推分支/合并后 Actions 总 run 仍 113、无新增 → workflow 收窄在远端生效。
> **结论不变**：runners 仍为 0、出站仍不通，没有任何一次 PR/push 被自动验证过，
> 本节所有「合并前本地实跑」与发版 `-race` 硬要求继续有效。详见
> `CHANGELOG.md` `[Unreleased]` 首条。

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
