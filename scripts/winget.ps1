#requires -Version 5.1
param([Parameter(Mandatory=$true)][ValidatePattern('^\d+\.\d+\.\d+$')][string]$Version)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$id = 'simiooo.squirrel_translate'
$out = Join-Path $root "dist\winget\manifests\s\simiooo\squirrel_translate\$Version"
New-Item -ItemType Directory -Path $out -Force | Out-Null
$base = "https://github.com/simiooo/squirrel_translate"
$utf8 = New-Object Text.UTF8Encoding($false)
$installer = @"
PackageIdentifier: $id
PackageVersion: $Version
InstallerType: nullsoft
Scope: user
InstallModes:
- interactive
- silent
UpgradeBehavior: install
ProductCode: io.github.simiooo.squirrel_translate
Installers:
"@
foreach ($arch in @('amd64', 'arm64')) {
    $name = "squirrel_translate-$Version-windows-$arch-setup.exe"
    $path = Join-Path $root "dist\release-$arch\$name"
    if (-not (Test-Path -LiteralPath $path)) { throw "Build the $arch release first." }
    $sha = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash
    $wingetArch = if ($arch -eq 'amd64') { 'x64' } else { 'arm64' }
    $installer += "`n- Architecture: $wingetArch`n  InstallerUrl: $base/releases/download/v$Version/$name`n  InstallerSha256: $sha"
}
$installer += "`nManifestType: installer`nManifestVersion: 1.6.0`n"
[IO.File]::WriteAllText((Join-Path $out "$id.installer.yaml"), $installer, $utf8)
[IO.File]::WriteAllText((Join-Path $out "$id.yaml"), @"
PackageIdentifier: $id
PackageVersion: $Version
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.6.0
"@, $utf8)
[IO.File]::WriteAllText((Join-Path $out "$id.locale.en-US.yaml"), @"
PackageIdentifier: $id
PackageVersion: $Version
PackageLocale: en-US
Publisher: simiooo
PublisherUrl: https://github.com/simiooo
PackageName: squirrel_translate
PackageUrl: $base
License: MIT
LicenseUrl: $base/blob/v$Version/LICENSE
PrivacyUrl: $base/blob/v$Version/PRIVACY.md
ShortDescription: Windows selection translation and vocabulary with your own LLM provider.
Tags:
- translation
- vocabulary
- llm
ManifestType: defaultLocale
ManifestVersion: 1.6.0
"@, $utf8)
# Unicode codepoints keep this script readable by Windows PowerShell 5.1 without a BOM.
$chineseName = -join ([char[]]@(0x677E,0x9F20,0x7FFB,0x8BD1))
[IO.File]::WriteAllText((Join-Path $out "$id.locale.zh-CN.yaml"), @"
PackageIdentifier: $id
PackageVersion: $Version
PackageLocale: zh-CN
Publisher: simiooo
PackageName: $chineseName
License: MIT
ShortDescription: Windows LLM translation and vocabulary tool.
ManifestType: locale
ManifestVersion: 1.6.0
"@, $utf8)
Write-Output $out
Write-Output 'Generated locally. Publish matching release assets, validate and test before submitting to winget-pkgs.'
