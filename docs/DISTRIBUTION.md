# 松鼠翻译 / squirrel_translate 分发方案

仓库：https://github.com/simiooo/squirrel_translate 。许可证：MIT。
本文件中的公开下载地址在首次正式发布之后才可用。当前清单只是待验证的提交材料，
不能据此认为 WinGet 已上架。

## 交付物

支持 Windows x64 和 ARM64。每个架构发布：

- `squirrel_translate-VERSION-windows-ARCH-setup.exe`：MyGo/NSIS 用户级安装器，
  写入开始菜单、Windows 已安装应用列表，带卸载程序；支持 `/S` 静默安装。
- `squirrel_translate-VERSION-windows-ARCH.zip`：免安装包，包含可执行文件、MIT、
  第三方许可证和隐私说明。不是“数据也放同目录”的便携模式；默认数据仍在 AppData。
- `SHA256SUMS-ARCH.txt`：发布资产的 SHA256。
- `install.ps1`：PowerShell 5.1 用户级安装/升级/卸载脚本，两种架构共用。

EXE、安装器、卸载器与快捷方式的图标由 `design/icon.png` 构建，运行时窗口、
任务栏、托盘与主标题使用相同图案。可编辑源为 OpenPencil 的 `design/app-icon.fig`；
SVG、1024px PNG 和 16/20/24/32/40/48/64/128/256px ICO 随源代码保存。
更新 OpenPencil 导出后运行 `go run ./cmd/icon-gen` 重新生成 ICO。

## 本地构建与发布

使用 go.mod 指定的 Go 版本。构建工具版本由 go.mod 锁定。无需 WebView 或 cgo。
首次构建 MyGo 会获取 NSIS；构建机需要网络，用户运行程序不需要 Go/NSIS。

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/release.ps1 -Version 0.1.0 -Architecture amd64
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/release.ps1 -Version 0.1.0 -Architecture arm64
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/test-install.ps1 -Version 0.1.0
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/winget.ps1 -Version 0.1.0
```

输出为 `dist/release-amd64`、`dist/release-arm64`、`dist/winget`。
两个 release.ps1 不要在同一 checkout 并发运行，因为会暂时写入 mygo.json 的版本。
本地脚本只用于开发验证，不作为正式发布包来源。正式发布包统一由 GitHub CI 构建。
GitHub Actions 在推送 `vX.Y.Z` 标签时执行测试、双架构构建，
创建 **draft release**；人工检查后在 GitHub 发布。手动触发工作流只产生 artifacts。
ARM64 目前只验证交叉编译，发布前还应在 ARM64 Windows 真机测试。

正式发布检查：安装、启动、托盘退出、快捷键、卸载、升级、两种界面语言、
旧配置/词汇表兼容，以及 Release 下载哈希与清单一致。不要覆盖已发布版本的资产；
修复后递增版本。当前未配置 Authenticode 签名，Windows 可能显示未知发布者；
后续可配置 MyGo 的 Windows signing（证书与密码仅放 CI Secret）。

## PowerShell 5.1

推荐先保存脚本再运行，方便检查；脚本使用 TLS 1.2、`-UseBasicParsing`、固定版本
GitHub Release URL，下载 ZIP 后核对同一版本 SHA256，再安装。SHA256 用于完整性校验，
不替代发布者签名或独立可信的哈希来源。以下命令须在首次公开 release 后使用：

```powershell
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
Invoke-WebRequest -UseBasicParsing 'https://github.com/simiooo/squirrel_translate/releases/latest/download/install.ps1' -OutFile install.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\install.ps1
# 固定版本 / 升级：先从托盘退出应用，再执行
powershell -NoProfile -ExecutionPolicy Bypass -File .\install.ps1 -Version 0.1.0
# 卸载（保留词汇、设置和凭据）
powershell -NoProfile -ExecutionPolicy Bypass -File .\install.ps1 -Uninstall
```

不修改全局 ExecutionPolicy 或 PATH。不需要管理员权限；默认安装至
`%LOCALAPPDATA%\Programs\squirrel_translate-portable`，创建开始菜单快捷方式。
支持 `-InstallDirectory`、`-NoShortcut` 和 `-Architecture amd64|arm64`。
自定义安装目录卸载时须传入相同路径。程序运行中拒绝覆盖，不强杀进程；
校验失败不修改安装；复制失败回滚已替换文件；只卸载安装器拥有的已知文件。
不接管已有未知目录，不删除无关文件。NSIS 与脚本安装是两种独立方式，避免混用。
ZIP 与校验文件已下载时可使用 `-ArchivePath`、`-ChecksumsPath`、`-Version` 离线安装。

从旧原型迁移时先退出旧程序；如果曾为旧原型启用开机启动，先在旧程序关闭该选项，
再在正式安装后的程序重新开启。正式包采用新的固定应用 ID；设置、词汇文件路径和
凭据存储服务名保持兼容，但旧可执行文件的启动项不会自动迁移。

其他 PowerShell 5.1 途径：直接下载 NSIS setup 后 `Start-Process -Wait -ArgumentList '/S'`，
或下载 ZIP 解压手动运行；若用户已有 Scoop，可在发布后增加自有 bucket 清单。
暂不同时维护 Scoop/Chocolatey 社区包，以免首次发布增加维护面。

## WinGet 调研结论（2026-10-21）

可提交。开源免费工具可以通过 `microsoft/winget-pkgs` 社区仓库分发；WinGet 存储清单，
实际安装包仍由项目 GitHub Releases 托管。首版选择 NSIS (`InstallerType: nullsoft`)，
可以直接沿用安装/卸载与快捷方式逻辑。候选 ID：`simiooo.squirrel_translate`，最终以
社区审核结果为准。提交前需公开可下载的固定版本安装器、准确 SHA256、产品元数据、
MIT 和隐私说明；目前脚本生成这些清单但不会提交 PR。

```powershell
# 生成后先校验（实际版本替换 0.1.0）
winget validate --manifest .\dist\winget\manifests\s\simiooo\squirrel_translate\0.1.0
# 在专用测试机/Windows Sandbox 启用 LocalManifestFiles，然后测试安装和卸载
winget install --manifest .\dist\winget\manifests\s\simiooo\squirrel_translate\0.1.0
# 审核合并且索引可用之后，普通用户才可运行：
winget install --id simiooo.squirrel_translate --exact
```

每个版本独立 PR，提交生成的四个 YAML 文件到清单路径。自动校验及人工审核可能要求
修改 ID、元数据或安装行为；不能保证审核通过时间。升级版本需重新生成哈希和清单。

官方依据：

- [提交流程](https://learn.microsoft.com/en-us/windows/package-manager/package/repository)
- [清单格式与 WinGetCreate](https://learn.microsoft.com/en-us/windows/package-manager/package/manifest)
- [本地验证与 Sandbox 测试](https://github.com/microsoft/winget-pkgs/blob/master/doc/README.md)
- [社区仓库政策](https://learn.microsoft.com/en-us/windows/package-manager/package/windows-package-manager-policies)
