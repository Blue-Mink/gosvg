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

CI（`.gitea/workflows/ci.yml` 与 `.github/workflows/ci.yml` **内容保持一致**）会跑同样检查，
外加 30 秒 `FuzzParse` 冒烟。改了任一 CI 文件请同步改另一个。

## 提交风格

Conventional Commits + 中文描述，沿用现有提交风格：`feat:` / `fix:` / `docs:` /
`chore:` / `test:`。一个提交一个主题，不要把文档、CI、代码混成一个巨型提交。

## 改了渲染逻辑

- 若**有意**改变渲染输出，重新生成基线：`go test -run TestGoldenRegression -update .`，
  并把 `testdata/golden.sha256` 的变更连同理由写进提交信息。
- 与 cairosvg 的对账（IoU / alphaAAD）目前在仓库外手工跑，数字更新请同步 `GOSVG_TODO.md`。

## 改了 fork 代码

`internal/rasterx`（MIT，srwiley/rasterx）与 `internal/vector`（BSD-3-Clause，x/image/vector）
是 fork，改动请同步更新同目录 `UPSTREAM.md`，并保留 `LICENSE`。

## 安全

- 提交前确认新增内容不含真实口令、token、内网凭据。
- 已知输入边界与报告方式见 `SECURITY.md`。
