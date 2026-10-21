# Squirrel Translate · 松鼠翻译

[简体中文](USER_GUIDE.md) · [Download releases](https://github.com/simiooo/squirrel_translate/releases) · [Privacy](../PRIVACY.md)

An open-source Windows selection translator. Translate selected text, use a keyboard shortcut, translate the clipboard, or type manually. Keep independent translation windows pinned and save words to your local vocabulary.

- Native desktop interface with light and dark themes.
- Simplified Chinese and English UI, selected manually or from your system locale.
- Streaming translations through your own Chat Completions-compatible model provider.
- Save, search, browse offline, and export word and phrase definitions.

## Install

Windows x64 and ARM64 are supported. Choose either installation method below.

### Option 1: Install from GitHub Releases

1. Open [Releases](https://github.com/simiooo/squirrel_translate/releases) and select the latest stable version.
2. Expand **Assets** and download the file for your computer:

| Computer | Installer (recommended) | Portable ZIP |
| --- | --- | --- |
| Intel / AMD 64-bit (x64) | `squirrel_translate-<version>-windows-amd64-setup.exe` | `squirrel_translate-<version>-windows-amd64.zip` |
| Windows ARM64 | `squirrel_translate-<version>-windows-arm64-setup.exe` | `squirrel_translate-<version>-windows-arm64.zip` |

3. Run `-setup.exe`, then open **Squirrel Translate** from the Start menu. For the ZIP, extract all files into the same folder and run `squirrel_translate.exe`.

Find your system type under Windows **Settings → System → About**. The **Source code** downloads are not runnable applications. Packages are currently unsigned, so Windows may show an unknown-publisher prompt.

### Option 2: Install with PowerShell

Open **Windows PowerShell 5.1** or later and paste these commands. The script detects your architecture, downloads the latest stable release, and verifies the ZIP's SHA-256 checksum:

```powershell
$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
$installer = Join-Path $env:TEMP 'squirrel_translate-install.ps1'
Invoke-WebRequest -UseBasicParsing 'https://github.com/simiooo/squirrel_translate/releases/latest/download/install.ps1' -OutFile $installer
powershell.exe -NoProfile -ExecutionPolicy Bypass -File $installer
```

The default location is `%LOCALAPPDATA%\Programs\squirrel_translate-portable`, with a Start menu shortcut. Administrator privileges are not required. The script does not change PATH; `ExecutionPolicy Bypass` applies only to the PowerShell process launched by this command.

After downloading the script, you can choose a version or location:

```powershell
$installer = Join-Path $env:TEMP 'squirrel_translate-install.ps1'
powershell.exe -NoProfile -ExecutionPolicy Bypass -File $installer -Version 0.1.0 -InstallDirectory "$env:LOCALAPPDATA\Programs\MySquirrelTranslate"
```

Only publicly released versions can be installed. Architecture detection is automatic; override it with `-Architecture amd64` or `-Architecture arm64` if needed. Updates discover existing installations from installer registration, the Start menu shortcut, and the default directory. If multiple installations exist, the script lists their paths; select one with `-InstallDirectory`. Custom locations without registration or a shortcut require an explicit path.

## First launch

1. Open **Settings → Model**.
2. Enter your provider's **base URL, model ID, and API key**. The provider must support a Chat Completions-compatible endpoint. Replace the example `https://your-provider.example/v1` with your actual base URL.
3. Click **Save**, then **Test connection**. This sends a real request and your provider may charge for it. An API key is not included with the app.
4. Choose your preferred and second languages under **Settings → Translate**. You can also select a target language when translating.

Without an API key, turn on **Demo** in the top-right corner to try the interface. Demo mode runs locally and displays sample content, not real translations.

### Change the UI language

Go to **Settings → Appearance → UI language**, choose system default, Simplified Chinese, or English, then **Save**. In Chinese, this is **设置 → 外观 → 界面语言 → 保存**.

The default follows your system: Chinese locales use Simplified Chinese; other locales use English. UI language and translation target languages are configured separately.

## Everyday use

| Action | How |
| --- | --- |
| Translate a selection | Select text in another app, then click Translate on the selection toolbar |
| Translate with a shortcut | Select text and press **Ctrl+Alt+T**; change it under Settings → Shortcuts |
| Translate the clipboard | Copy text, then choose clipboard translation from the tray menu |
| Type manually | Enter or paste text in the main window and click Translate |
| Keep a translation window | Pin it; later selections will not replace its content |
| Save a word or phrase | Wait for the definition to finish, then click the star; use Vocabulary to search, view, remove, or export JSON |
| Pause automatic selection | Use the pause option in the tray menu |
| Exit completely | Choose Quit from the tray menu; closing the main window hides it to the tray |

Translation windows support copy, stop, and retranslate. Unpinned windows close when you leave them and cancel unfinished requests. If automatic classification is incorrect, switch to word, phrase, or sentence / paragraph mode. Sentence translations cannot be saved as vocabulary entries.

## Update and uninstall

Choose **Check for updates** under **Settings → Updates** or in the tray menu. When a newer version is available, click **Download update and restart**. The app downloads the stable release for your architecture, verifies SHA-256, quits, replaces its files and restarts. Installer, PowerShell and portable ZIP installations keep their location, settings, vocabulary and API key.

Automatic checks run daily and only notify; they never install without your action. You can disable them on the Updates page. Network or verification failures leave the current version intact. Replacement or startup failures trigger an attempt to restore the previous version. If the installation directory is not writable, use a writable location or update manually using your original installation method.

For older versions without the update entry, or when updating manually, **quit from the tray** first:

- **Installer:** Download and run the newer installer from Releases. Uninstall through Windows Settings → Apps.
- **ZIP:** Download and extract the newer version, then quit the old app before using it. Delete the extracted program folder to uninstall.
- **PowerShell:** Run the full installation commands above again to update to the latest stable version. To uninstall, use the downloaded script; if your temporary files were cleared, repeat the download step first:

```powershell
$installer = Join-Path $env:TEMP 'squirrel_translate-install.ps1'
powershell.exe -NoProfile -ExecutionPolicy Bypass -File $installer -Uninstall
```

The PowerShell script can also upgrade installer and recognized portable ZIP installations in place. It preserves the installation type, settings, existing shortcuts (including custom arguments), and uninstaller. Verification failures leave the old version intact. Replacement failures restore backups; if recovery fails too, backups are retained and their path is shown. Quit the running app from the tray first. Uninstall installer editions through Windows Settings, remove ZIP application files manually, and use `-Uninstall` for script installations (add `-InstallDirectory` if discovery cannot find them). Removing the program preserves settings, vocabulary, and the saved API key.

## Troubleshooting

**Nothing happens when selecting text?** Some apps do not expose readable selections. Images, scanned PDFs, custom controls, remote desktops, and elevated windows may not work. Copy the text and explicitly translate the clipboard, or type it manually. OCR is not supported yet.

**Shortcut does not work?** Check for conflicts with other apps and change it under Settings → Shortcuts.

**Model connection fails?** Check your saved URL, model ID, API key, and network. Only Chat Completions-compatible endpoints are supported. If your provider rejects structured output, switch its mode back to `prompt`. Leave other model parameters empty to use provider defaults.

**Can I install with WinGet?** The package is not listed yet. Use Releases or PowerShell for now.

## Data and privacy

Settings and vocabulary are stored in `%APPDATA%\SquirrelTranslate`. API keys are stored in Windows Credential Manager. Use Vocabulary's JSON export to back up saved entries.

When translating, selected text and any enabled context are sent to your configured model provider. Selection detection alone does not send model requests. Ordinary translation history is not saved by default. See [Privacy](../PRIVACY.md). Model-generated definitions can be inaccurate.

## Open source

Licensed under [MIT](../LICENSE). Third-party dependencies retain their own licenses. Report problems through [GitHub Issues](https://github.com/simiooo/squirrel_translate/issues).

## Pronunciation and reading aloud

Word translations show a phonetic transcription when the model can provide one. Saving a word preserves it in the vocabulary list and detail view. Click the speaker next to a word to read it aloud, and click again to stop. Starting another reading, retranslating, or closing its floating window also stops playback.

Reading uses Windows voices installed on your computer, with no additional speech API or internet connection. If a voice is missing, add speech for that language in Windows Settings. Transcriptions come from your translation model; system voices may use a different accent.

Existing entries are not queried automatically. Click **Add pronunciation** in the vocabulary detail view, then save the completed result to update the entry. This uses your configured translation model.
