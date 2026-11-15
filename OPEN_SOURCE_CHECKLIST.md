# 开源收尾清单（推送到外部 Git 之前）

> 当前约束：**不推外部 Git**，源码只托管在内网 Gitea
> `https://github.com/Blue-Mink/gosvg`。本清单记录将来解除约束后要逐项做的事。

- [ ] **确认目标仓**：`github.com/Blue-Mink/gosvg` 是否已存在、是否为空仓。
      `go.mod` 的 module 路径就是它，改路径属于破坏性变更（所有 import 要改），
      所以优先让 GitHub 仓名与 module 对齐，而不是改 module。
- [x] **全历史敏感串扫描**：逐 blob 检查口令 / token / 内网凭据 / 私人邮箱。
      （本项目历史上别的仓库踩过"HEAD 洗干净了、历史 blob 还躺着"的坑。）
      提交作者邮箱目前是 `Blue-Mink@users.noreply.github.com`，无需处理。
      ✅ 2026-11-10 已扫 101 个提交 / 19337 行 diff：40 位 hex（排除 commit hash 后）、
      私钥、口令赋值、bearer、apikey **全部 0 命中**。
- [ ] **私有地址清理**：README / SECURITY / TODO 里的
      `https://github.com/Blue-Mink/...` 改为公开 URL；保留在内网文档里不算泄露，
      但公开仓里应换成实际地址。
- [x] **许可证核对**：自研 MIT；`internal/rasterx`（MIT）、`internal/vector`（BSD-3-Clause）
      的 LICENSE 与 `UPSTREAM.md` 齐全。
      ✅ 2026-11-10 核实 `git ls-files`：`LICENSE`、`internal/vector/LICENSE`、
      `internal/rasterx/LICENSE` 三份均在，两份 `UPSTREAM.md` 的归属声明一致。
- [ ] **CI 徽章**：README 顶部换成正经的 GitHub Actions badge（当前只有静态 Go/License/deps 徽章）。
- [x] **首个 tag**：内网 Gitea 已打 `v0.1.0` annotated tag（PR #1 合并后）。
- [x] **Release**：内网 Gitea 已发 `v0.2.0` / `v0.2.1` / `v0.2.2` / `v0.2.3` /
      `v0.2.4` / `v0.3.0` Release（notes 以 `CHANGELOG.md` 为准；v0.2.3 起 tag
      均由 Release API 自建，勿先 push tag）。`go get github.com/Blue-Mink/gosvg@latest`
      仍须在推送
      外部 Git 之后才能验证，见下项。
- [ ] **加 remote 并推送**：`git remote add github <url>` 后 `git push github main --tags`；
      **不要**把 GitHub 设为默认 `origin`，避免内网日常开发误推。
- [ ] **验证 outside view**：用全新临时 `GOPATH` 跑一次 `go get` + `go test`，
      确认不依赖内网资源。

## 触发条件

用户明确解除「不推外部 Git」约束后，按上面顺序执行；在此之前本文件只作为待办留存，
不产生任何对外动作。
