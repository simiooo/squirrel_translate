# squirrel_translate · 松鼠翻译

[中文说明](README.md) · [Distribution guide](docs/DISTRIBUTION.md) · [Privacy](PRIVACY.md)

A native Windows selection translator and vocabulary tool, built with Go and MyGo.
Use an OpenAI-compatible LLM endpoint of your choice. No WebView is required.

Select text to open the translation toolbar, or press **Ctrl+Alt+T**. Each floating
translation owns its selection. Pin it to keep it open; closing an unpinned window
cancels its request. Save word and phrase definitions to your local vocabulary.
Reasoning is shown only as a temporary loading line and disappears when the answer starts.

UI languages: **Simplified Chinese and English**. Choose Settings → Appearance → UI
language, then Save. The default follows Windows (Chinese locales use Simplified Chinese;
other locales use English). UI language is independent of translation languages.

Download a Windows x64 or ARM64 installer/ZIP from
[Releases](https://github.com/simiooo/squirrel_translate/releases) after the first release
is published. WinGet submission is prepared, but the package is not yet listed.
The PowerShell 5.1 installer supports checksums, upgrades and uninstall; see the distribution guide.

For development, use the Go version in go.mod:

```powershell
go test ./...
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/build.ps1
.\dist\windows-amd64\squirrel_translate.exe --demo --lang en
```

Configure the endpoint, model and API key in Settings. Save before testing the connection.
Demo mode runs offline. API keys use Windows Credential Manager; settings and vocabulary
remain in `%APPDATA%\SquirrelTranslate`. Existing data remains compatible.

The app and original icon are MIT licensed. Bundled dependencies retain their own licenses,
collected in THIRD_PARTY_NOTICES.txt during builds.
