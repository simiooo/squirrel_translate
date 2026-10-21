# squirrel_translate · 松鼠翻译

[简体中文](README.md)

A Windows selection translator with Chinese and English UI, floating translations, and saved vocabulary.

## Install

**[Download from Releases](https://github.com/simiooo/squirrel_translate/releases)** → Run the `-setup.exe` installer. Choose **amd64** for Intel / AMD PCs or **arm64** for Windows ARM PCs. A portable ZIP is also available.

Or run one command in **PowerShell 5.1+** to download and install automatically:

```powershell
irm https://github.com/simiooo/squirrel_translate/releases/latest/download/install.ps1 | iex
```

Open **squirrel_translate** from the Start menu. To update, quit from the tray and run the same command again.

> v0.1.0 is still a draft. Downloads and installation become available after publication.

## First use

1. Open **Settings → Model** and enter your provider's **API base URL, model ID, and API key** (Chat Completions-compatible).
2. Click **Save → Test connection**. No key? Turn on **Demo** in the top-right corner to try it.
3. Select text in another app and click Translate, or press **Ctrl+Alt+T**.

**Pin** a translation window to keep it open; click **☆** to save a word or phrase.
Change the UI language under **Settings → Appearance → UI language**, then save.

[More help](docs/USER_GUIDE.en.md) · [Privacy](PRIVACY.md) · [Report an issue](https://github.com/simiooo/squirrel_translate/issues) · [MIT](LICENSE)
