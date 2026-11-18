# 开源收尾清单（发布到 GitHub 之前）

> 目标仓：GitHub `https://github.com/Blue-Mink/gosvg`（Private 仓库），
> 与 `go.mod` 的 module 路径一致。本清单记录发布前的准备事项与核对结果。

- [ ] **确认目标仓**：`github.com/Blue-Mink/gosvg` 是否已存在、是否为空仓。
      `go.mod` 的 module 路径就是它，改路径属于破坏性变更（所有 import 要改），
      所以优先让 GitHub 仓名与 module 对齐，而不是改 module。
- [x] **全历史敏感串扫描**：逐 blob 检查口令 / token / 内网凭据 / 私人邮箱。
      （本项目历史上别的仓库踩过"HEAD 洗干净了、历史 blob 还躺着"的坑。）
      提交作者邮箱目前是 `Blue-Mink@users.noreply.github.com`，无需处理。
      ✅ 2026-11-10 已扫 101 个提交 / 19337 行 diff：40 位 hex（排除 commit hash 后）、
      私钥、口令赋值、bearer、apikey **全部 0 命中**。
- [x] **私有地址清理**：README / SECURITY / 本清单里的内网地址已全部改为
      GitHub 公开 URL；历史提交作者邮箱（46 个私人 QQ 邮箱）已通过
      `git filter-branch` 全量改写为 `Blue-Mink@users.noreply.github.com`，
      并 `git gc --prune=now` 清除旧对象。全历史复扫内网 IP / 私人邮箱 0 命中。
- [x] **许可证核对**：自研 MIT；`internal/rasterx`（MIT）、`internal/vector`（BSD-3-Clause）
      的 LICENSE 与 `UPSTREAM.md` 齐全。
      ✅ 2026-11-10 核实 `git ls-files`：`LICENSE`、`internal/vector/LICENSE`、
      `internal/rasterx/LICENSE` 三份均在，两份 `UPSTREAM.md` 的归属声明一致。
- [x] **CI 徽章**：README 顶部已加 GitHub Actions badge；两份 workflow
      （`.gitea`/`.github`）内容逐字一致，恢复 `push`/`pull_request` 自动触发。
- [x] **tag**：内网 Gitea 已打 `v0.1.0` 起全套 annotated tag，最新 `v0.4.1`。
- [x] **Release**：已发 `v0.2.0` → `v0.4.1` Release（notes 以 `CHANGELOG.md` 为准）。
      历史写操作中 tag 由 Release API 自建。历史已改写（脱敏），GitHub 侧 tag/Release
      须在推送后按改写后的 commit 重建。
- [ ] **加 remote 并推送**：在有 GitHub 出站访问权限的机器上执行
      `git remote add github https://github.com/Blue-Mink/gosvg.git` 后
      `git push github main --tags`；**不要**把 GitHub 设为默认 `origin`。
- [ ] **验证 outside view**：用全新临时 `GOPATH` 跑一次 `go get` + `go test`，
      确认不依赖内网资源。

## 触发条件

本机 GitHub 出站不通（实测 github.com / api.github.com 均连接失败），需在具备
出站访问与凭据的机器上完成「加 remote 并推送」及 GitHub 侧 tag/Release 创建；
在此之前本文件记录准备状态，不产生对外动作。
