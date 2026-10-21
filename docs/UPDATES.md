# 应用内更新

入口：设置 → 更新、托盘 → 检查更新。默认每 24 小时检查，首次在启动 10 秒后检查；只发送通知。偏好与最近成功检查时间保存在用户数据目录的 `updates.json`，不改变模型配置。

`internal/update` 使用 [GitHub latest release API](https://docs.github.com/en/rest/releases/releases#get-the-latest-release)。只接受 `vMAJOR.MINOR.PATCH` 正式发布，按数值比较版本，不降级。按运行中程序的 amd64/arm64 架构选择现有 ZIP 和 `SHA256SUMS-<arch>.txt`，不需要额外发布资产。官方包仍由 GitHub Actions 构建，不上传本地构建结果。

## 替换流程

1. 用户点击“下载更新并重启”。在程序旁创建 `.squirrel-update-*` 暂存目录，提前验证写入权限。下载有超时、取消及大小限制，初始下载地址必须匹配本仓库对应版本和资产，重定向必须保持 HTTPS。
2. 校验 SHA-256、资产大小、ZIP 固定文件列表及 Windows PE 架构。拒绝路径穿越、重复条目、链接和错误架构。不写用户数据目录或凭据。
3. 把当前程序复制为暂存目录中的 `helper.exe`，启动 `--apply-update`。辅助进程验证计划和父进程实际路径，取得父进程句柄并发出就绪信号，应用收到后才正常退出并取消翻译请求。
4. 辅助进程等待父进程退出，再次校验 ZIP，备份四个发布文件并替换。备份和安装目录处于同一卷；覆盖失败会逆序恢复。
5. 用原参数和原路径重启。新进程完成窗口初始化后回传就绪；30 秒内未就绪或提前退出会终止该子进程、回滚并尝试重启旧版。旧版恢复失败时保留暂存目录与 `result.json` 供人工恢复。
6. 重启成功的应用在辅助进程结束后清理暂存目录。更新结果显示在应用状态与更新页。

## 安装方式

- ZIP：保持当前实际 exe 文件名与路径，允许重命名，不改动目录内其他文件。
- PowerShell：验证 `.squirrel-install.json` 的仓库和目录归属，更新版本/架构并保留快捷方式字段。记录随文件回滚。
- NSIS 安装器：将 ZIP 中的程序映射到正在运行的 `Squirrel Translate.exe`。只有 HKCU 卸载项的 InstallLocation 和 DisplayIcon 同时匹配时才更新 DisplayVersion；保留卸载器、现有快捷方式与开机启动路径。当前安装器为用户级安装。未来若修改发布文件列表或卸载结构，须同步升级此策略。
- 不可写目录：在退出前报错，不自行提权或转移安装。用户可通过原安装方式更新。

SHA-256 来自同一 GitHub Release，提供完整性校验；当前发行未做代码签名。首次获得本功能仍需从 Releases/原安装方式更新一次，不带此功能的旧二进制不会自行增加更新入口。

## 验证

`go test ./internal/update` 包含真正的 Windows 子进程退出、exe 替换、就绪握手及重启，使用临时中文/空格路径和独立注册表测试项；覆盖三种安装布局、重启失败回滚、文件占用、损坏下载、取消、非法归属、重复资产、ZIP 路径和架构拒绝。不会触及用户已有安装或模型凭据。

可选真实下载验证（仅临时目录，绝不安装）：

```powershell
$env:SQUIRREL_TEST_GITHUB = '1'
go test ./internal/update -run TestPublishedReleaseDownload -v -count=1
Remove-Item Env:SQUIRREL_TEST_GITHUB
```

本地开发 `go build` 没有包版本时不允许应用内更新；正式版本取自 MyGo 的打包元数据。`versionOverride` 仅供隔离开发验证用，不用于正式发布版本号。

## PowerShell installer upgrades

`scripts/install.ps1` discovers current-user NSIS registration (both registry layouts), the standard Start menu shortcut, and the default script directory. An explicit `-InstallDirectory` takes precedence; multiple discovered installations require an explicit selection. Script ownership markers identify PowerShell installs; recognized product metadata identifies portable ZIP executables. Unregistered custom locations without a shortcut cannot be discovered automatically.

The script keeps NSIS executable naming, uninstall registration and uninstaller, updating only DisplayVersion. Portable installs remain portable. Existing shortcut bytes and unknown configuration files are preserved. Downloads and checksums are validated before changes. Each existing file is renamed before replacement so a locked file cannot be truncated; partial failures restore backed-up files and metadata. Recovery failure retains the backup directory and reports it. This is exception recovery, not a guarantee against power loss or process termination; unlike the in-app updater, the script does not launch or health-check the new executable.

Run `scripts/test-install.ps1 -Version <version> -Architecture <arch>` and `scripts/test-install-discovery.ps1 -Version <version> -Architecture <arch>` after packaging. Discovery tests isolate registry keys, shortcuts and directories, exercising script/NSIS/portable installs, ambiguity, explicit selection, data and shortcut preservation, locked-file failure and injected metadata failure. Both tests run in the release workflow for both architectures; ARM64 files can be replaced on x64 without executing them.
