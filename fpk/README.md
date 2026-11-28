# GoSvg FPK 构建源（fnOS / 飞牛 OS）

本目录是 [gosvg](../README.md) CLI 工具集的 **fnOS FPK 应用打包源**。gosvg 本体是纯 Go
的 SVG 渲染命令行工具，没有 Web UI、也没有常驻服务；打包成 FPK 只是为了能在飞牛 OS 的
应用中心里安装/启动/停止。真正干活的是 4 个 CLI，安装后位于 `${TRIM_APPDEST}/bin/`，
通过 SSH 调用。

## 目录结构

```
fpk/
├── manifest                 # 应用身份信息（appname/version/display_name/desc/...）
├── config/
│   ├── privilege            # run-as=package，username/groupname=gosvg
│   └── resource             # 无 UI/服务/docker，空对象
├── cmd/
│   ├── main                 # 轻量常驻哨兵：sleep infinity，仅满足生命周期模型
│   └── *_init / *_callback  # 安装/卸载/升级/配置回调（官方骨架最小实现）
├── app/ui/
│   ├── index.cgi            # 文档页（gosvg 使用说明 + FAQ，CGI 直出静态 HTML）
│   ├── config               # UI 配置
│   └── images/              # 图标资源
├── icon.svg                 # 应用图标源文件
└── ICON*.PNG                # 打包用 PNG 图标
```

## 构建方法

需要 [fnOS fnpack](https://github.com/fn-pack/fnpack)（`fpk.py`）以及 Go 1.27+。

1. **编译 4 个 CLI**（静态链接，`CGO=0`），分别产出 amd64 / arm64：

   ```sh
   for arch in amd64 arm64; do
     for tool in render verify corpus iou; do
       CGO_ENABLED=0 GOOS=linux GOARCH=$arch \
         go build -o overlay-$arch/app/bin/gosvg-$tool ./cmd/$tool
     done
   done
   ```

   `overlay-<arch>/app/bin/` 里的 4 个二进制会被覆盖到模板树的 `app/bin/` 下。

2. **架构无关源**就是本目录（`manifest` / `config/` / `cmd/` / `app/ui/` / 图标）。
   注意 `manifest` 里的 `platform` 由打包工具按架构重写，无需手动改。

3. **打包 + 审计**（用 `fpk.py`，不要用 `fnpack` 覆盖安装已装应用）：

   ```sh
   fpk.py doctor --project fpk
   fpk.py build  --project fpk --out dist --arch both \
     --overlay-amd64 overlay-amd64 --overlay-arm64 overlay-arm64
   fpk.py inspect dist/gosvg-<version>-fnos-amd64.fpk
   fpk.py inspect dist/gosvg-<version>-fnos-arm64.fpk
   ```

4. **部署**：在目标 NAS 上先 `appcenter-cli stop gosvg` → `uninstall gosvg`（确认
   `noinstall`）→ 上传 `.fpk` → `install-fpk <pkg>` → `start gosvg`。升级走生命周期替换，
   不要 FPK over FPK。

## 哨兵说明

`cmd/main` 里的 `start` 用 `exec sleep infinity` 起一个常驻空转进程，PID 写入
`${TRIM_PKGVAR}/app.pid`，让应用中心显示“运行中”。它**不提供任何服务**，只为了满足 FPK
的 `start/stop/status` 生命周期模型；CLI 由用户按需 SSH 调用。文档页 `app/ui/index.cgi`
也通过读取该状态给出使用提示。

## 版本

构建源版本与 gosvg 主仓 Release 版本保持一致（当前 `0.5.0`）。发布时同步更新
`manifest` 的 `version` 与本文件，并重新打包双架构。
