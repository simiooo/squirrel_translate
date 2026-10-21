# 松鼠翻译 · squirrel_translate

[English](README.en.md)

Windows 划词翻译工具，支持中英文界面、翻译浮窗和单词收藏。

## 安装

### 方式一：下载安装

[前往 Releases 下载](https://github.com/simiooo/squirrel_translate/releases)

### 方式二：通过 PowerShell 安装

```powershell
irm https://github.com/simiooo/squirrel_translate/releases/latest/download/install.ps1 | iex
```

安装后从开始菜单打开 **squirrel_translate**。更新时先从托盘退出，再运行同一条命令。

> v0.1.0 尚为草稿，公开发布后即可下载和安装。

## 首次使用

1. 打开 **设置 → 模型**，填写服务商的 **API 地址、模型 ID 和 API Key**（支持 Chat Completions 兼容接口）。
2. 点击 **保存 → 测试连接**。没有 Key？打开右上角 **演示** 即可体验。
3. 在其他应用中选中文字，点击翻译，或按 **Ctrl+Alt+T**。

点击 **置顶** 保留翻译浮窗，点击 **☆** 收藏单词或短语。
在 **设置 → 外观 → 界面语言** 切换中文 / English，保存生效。

[更多使用帮助](docs/USER_GUIDE.md) · [隐私](PRIVACY.md) · [反馈问题](https://github.com/simiooo/squirrel_translate/issues) · [MIT](LICENSE)
