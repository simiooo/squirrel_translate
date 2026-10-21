# Privacy / 隐私说明

松鼠翻译（Squirrel Translate）在本机保存设置和词汇表，默认目录为
`%APPDATA%\SquirrelTranslate`。API Key 通过 Windows 凭据存储保存，不写入配置文件。

翻译时，选中的文本、目标语言以及启用时的上下文会发送至用户配置的模型服务。
该服务如何保存或处理请求取决于其隐私政策。应用不提供自有的翻译中转服务，
不添加遥测或广告。演示模式不发送模型请求。全局输入监听只用于判断划词操作，
不是键盘内容记录器。卸载默认保留设置、词汇表及凭据，便于重装。

正式版默认每天向 GitHub 检查一次新版本，可在「设置 → 更新」关闭。更新检查和下载不发送选文、词汇表、模型配置或 API Key；GitHub 会收到常规网络连接信息。只有用户点击更新后才下载并安装。

Settings and vocabulary are stored locally in `%APPDATA%\SquirrelTranslate`.
API keys use the Windows credential store, not the settings file. Translation sends
selected text, target language and optional context to the model endpoint you configure.
That provider's privacy policy applies. The app adds no telemetry, advertising or
translation relay. Demo mode makes no model requests. Global input monitoring detects
selection gestures; it does not log typed text. Uninstall preserves user data and credentials.

Published builds check GitHub for updates daily by default; disable this under Settings → Updates.
Update requests do not send selected text, vocabulary, model settings or API keys. GitHub receives
ordinary network connection information. Downloads and installation start only after you choose to update.
