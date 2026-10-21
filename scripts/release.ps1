#requires -Version 5.1
param([Parameter(Mandatory=$true)][ValidatePattern('^\d+\.\d+\.\d+$')][string]$Version,
      [ValidateSet('amd64','arm64')][string]$Architecture = 'amd64')
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$configPath = Join-Path $root 'mygo.json'
$original = [IO.File]::ReadAllText($configPath)
try {
    $config = $original | ConvertFrom-Json
    $config.version = $Version
    [IO.File]::WriteAllText($configPath, ($config | ConvertTo-Json -Depth 10), (New-Object Text.UTF8Encoding($false)))
    & (Join-Path $PSScriptRoot 'build.ps1') -Architecture $Architecture
    $built = Join-Path $root "dist\windows-$Architecture"
    $out = Join-Path $root "dist\release-$Architecture"
    New-Item -ItemType Directory -Path $out -Force | Out-Null
    $zipName = "squirrel_translate-$Version-windows-$Architecture.zip"
    $zipPath = Join-Path $out $zipName
    $files = @('squirrel_translate.exe','LICENSE','THIRD_PARTY_NOTICES.txt','PRIVACY.md') | ForEach-Object { Join-Path $built $_ }
    Compress-Archive -LiteralPath $files -DestinationPath $zipPath -Force
    $setupName = "squirrel_translate-$Version-windows-$Architecture-setup.exe"
    Copy-Item -LiteralPath (Join-Path $built "squirrel_translate Setup $Version $Architecture.exe") -Destination (Join-Path $out $setupName)
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'install.ps1') -Destination (Join-Path $out 'install.ps1')
    $checksums = @($zipName, $setupName, 'install.ps1') | ForEach-Object {
        $hash = (Get-FileHash -LiteralPath (Join-Path $out $_) -Algorithm SHA256).Hash.ToLowerInvariant()
        "$hash  $_"
    }
    [IO.File]::WriteAllLines((Join-Path $out "SHA256SUMS-$Architecture.txt"), $checksums, (New-Object Text.UTF8Encoding($false)))
    Write-Output $out
} finally { [IO.File]::WriteAllText($configPath, $original, (New-Object Text.UTF8Encoding($false))) }
