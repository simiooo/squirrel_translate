#requires -Version 5.1
param([ValidateSet('amd64', 'arm64')][string]$Architecture = 'amd64')
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
Push-Location -LiteralPath $projectRoot
try {
    & (Join-Path $PSScriptRoot 'notices.ps1')
    go run github.com/egoist/mygo/cmd/mygo build -platform "windows/$Architecture"
    if ($LASTEXITCODE -ne 0) { throw 'Build failed.' }
    # Keep the portable executable's technical filename stable. NSIS uses the display name.
    Move-Item -LiteralPath (Join-Path $projectRoot "dist\windows-$Architecture\Squirrel Translate.exe") -Destination (Join-Path $projectRoot "dist\windows-$Architecture\squirrel_translate.exe") -Force
    Write-Output (Join-Path $projectRoot "dist\windows-$Architecture\squirrel_translate.exe")
}
finally {
    Pop-Location
}
