# 松鼠翻译 · Squirrel Translate

[English](USER_GUIDE.en.md) · [下载 Releases](https://github.com/simiooo/squirrel_translate/releases) · [隐私说明](../PRIVACY.md)

松鼠翻译是一款开源 Windows 划词翻译工具。选中文字即可翻译，也可以使用快捷键、剪贴板或手动输入；支持独立翻译浮窗、置顶和本地词汇收藏。

- 原生桌面界面，支持浅色、深色主题。
- 简体中文、英文界面，可跟随系统或手动选择。
- 接入你自己的 Chat Completions 兼容模型服务，流式显示译文。
- 单词和短语可收藏、搜索、离线查看与导出。

## 安装

支持 Windows x64 与 ARM64。以下任选一种安装方式。

> v0.1.0 当前为 Release 草稿。维护者公开发布后，Release 下载及下面的 PowerShell 安装命令才可用。

### 方式一：从 Release 页面安装

1. 打开 [Releases 页面](https://github.com/simiooo/squirrel_translate/releases)，选择最新正式版本。
2. 展开 **Assets**，根据电脑架构下载文件：

| 电脑类型 | 安装程序（推荐） | 免安装 ZIP |
| --- | --- | --- |
| Intel / AMD 64 位（x64） | `squirrel_translate-<版本>-windows-amd64-setup.exe` | `squirrel_translate-<版本>-windows-amd64.zip` |
| Windows ARM64 | `squirrel_translate-<版本>-windows-arm64-setup.exe` | `squirrel_translate-<版本>-windows-arm64.zip` |

3. 运行 `-setup.exe` 完成安装，然后从开始菜单打开 **Squirrel Translate**。使用 ZIP 时，将全部文件解压到同一文件夹，运行 `squirrel_translate.exe`。

可在 Windows「设置 → 系统 → 系统信息」查看系统类型。Release 中的 **Source code** 是源码，不是可直接运行的应用。当前发布包尚未代码签名，Windows 可能显示发布者未知的提示。

### 方式二：PowerShell 安装

打开 **Windows PowerShell 5.1** 或更新版本，粘贴以下命令。脚本会自动选择架构，下载最新正式版并校验 ZIP 的 SHA-256：

```powershell
$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
$installer = Join-Path $env:TEMP 'squirrel_translate-install.ps1'
Invoke-WebRequest -UseBasicParsing 'https://github.com/simiooo/squirrel_translate/releases/latest/download/install.ps1' -OutFile $installer
powershell.exe -NoProfile -ExecutionPolicy Bypass -File $installer
```

默认安装到 `%LOCALAPPDATA%\Programs\squirrel_translate-portable`，并创建开始菜单快捷方式。无需管理员权限，不修改 PATH；`ExecutionPolicy Bypass` 仅作用于本次启动的 PowerShell 进程。

下载脚本后，也可以指定版本或安装位置，例如：

```powershell
$installer = Join-Path $env:TEMP 'squirrel_translate-install.ps1'
powershell.exe -NoProfile -ExecutionPolicy Bypass -File $installer -Version 0.1.0 -InstallDirectory "$env:LOCALAPPDATA\Programs\MySquirrelTranslate"
```

仅可安装已公开发布的版本。默认自动识别架构，必要时添加 `-Architecture amd64` 或 `-Architecture arm64`。自定义安装目录后，升级和卸载必须使用相同的 `-InstallDirectory`。

## 首次使用

1. 打开应用，进入 **设置 → 模型**。
2. 填写服务商提供的 **基础地址、模型 ID 和 API Key**。需要支持 Chat Completions 兼容接口；基础地址示例为 `https://your-provider.example/v1`，请替换成实际地址。
3. 点击 **保存**，再点击 **测试连接**。连接测试会发送真实请求，服务商可能计费。应用不附带 API Key。
4. 在 **设置 → 翻译** 中选择首选语言和第二语言；翻译时也可手动指定目标语言。

没有 API Key 时，可以打开右上角的 **演示** 开关体验界面。演示在本地运行，显示的是示例内容，不是真实翻译。

### 切换界面语言

进入 **设置 → 外观 → 界面语言**，选择「跟随系统」「简体中文」或「English」，然后保存。英文界面对应 **Settings → Appearance → UI language → Save**。

默认跟随系统：中文区域使用简体中文，其他区域使用英文。界面语言与翻译目标语言分别设置。

## 日常使用

| 操作 | 方法 |
| --- | --- |
| 划词翻译 | 在其他应用中选中文字，点击出现的工具条中的翻译按钮 |
| 快捷键翻译 | 选中文字后按 **Ctrl+Alt+T**；可在「设置 → 快捷键」修改 |
| 剪贴板翻译 | 先复制文字，再从托盘菜单选择剪贴板翻译 |
| 手动输入 | 在主窗口输入或粘贴文字，点击翻译 |
| 保留翻译浮窗 | 点击置顶；后续选文不会覆盖已置顶窗口 |
| 收藏单词或短语 | 等待释义完成后点击星星，在「词汇表」搜索、查看、移除或导出 JSON |
| 暂停自动划词 | 使用托盘菜单中的暂停选项 |
| 完全退出 | 在托盘菜单选择退出；关闭主窗口只会隐藏到托盘 |

翻译浮窗支持复制、停止和重新翻译。未置顶的浮窗在离开后关闭，并取消未完成的请求。自动模式识别不准确时，可切换为单词、短语或句子 / 段落模式；句子不提供词汇收藏。

## 更新与卸载

更新前，请先从托盘**退出**应用。当前没有应用内自动更新。

- **安装程序**：从 Releases 下载新版安装程序并运行；卸载使用 Windows「设置 → 应用」。
- **ZIP**：下载新版并解压，退出旧程序后使用新版；卸载时删除解压的程序文件夹。
- **PowerShell**：重新运行上面的完整安装命令即可更新到最新正式版。卸载时使用下载过的脚本（临时文件已清理时，先重新执行上面的下载步骤）：

```powershell
$installer = Join-Path $env:TEMP 'squirrel_translate-install.ps1'
powershell.exe -NoProfile -ExecutionPolicy Bypass -File $installer -Uninstall
```

PowerShell 脚本和安装程序使用不同安装目录，请沿用原来的方式更新和卸载。卸载程序文件会保留配置、词汇表和保存的 API Key。

## 常见问题

**选中文字没有反应？** 部分应用不提供可读取的选文接口；图片、扫描 PDF、自绘控件、远程桌面及高权限窗口可能无法取词。可改用复制后主动翻译剪贴板，或手动输入。目前不支持 OCR。

**快捷键无效？** 检查是否与其他软件冲突，在「设置 → 快捷键」更换组合。

**模型连接失败？** 检查保存后的地址、模型 ID、API Key 与网络。只支持 Chat Completions 兼容接口；如果服务商不支持结构化输出模式，可将其改回 `prompt`。其他模型参数可留空使用服务商默认值。

**可以用 WinGet 安装吗？** 尚未上架。当前请使用 Release 页面或 PowerShell 安装。

## 数据与隐私

配置和词汇表保存在 `%APPDATA%\SquirrelTranslate`，API Key 保存于 Windows Credential Manager。备份词汇可使用词汇表的 JSON 导出功能。

翻译时，选文和启用的上下文会发送给你配置的模型服务商。自动划词阶段不发送模型请求；普通翻译历史默认不保存。详见 [隐私说明](../PRIVACY.md)。模型生成的释义可能有误。

## 开源

采用 [MIT 许可证](../LICENSE)，第三方依赖保留各自许可证。反馈问题请前往 [GitHub Issues](https://github.com/simiooo/squirrel_translate/issues)。
