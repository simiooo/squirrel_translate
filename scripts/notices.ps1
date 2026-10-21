#requires -Version 5.1
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Push-Location -LiteralPath $root
try {
    go mod download
    if ($LASTEXITCODE -ne 0) { throw 'Dependency download failed.' }
    $modules = @(go list -deps -f '{{if .Module}}{{.Module.Path}}|{{.Module.Dir}}{{end}}' ./cmd/squirrel | Where-Object { $_ } | Sort-Object -Unique)
    if ($LASTEXITCODE -ne 0) { throw 'Dependency inventory failed.' }
    $goRoot = go env GOROOT
    $text = New-Object System.Text.StringBuilder
    [void]$text.AppendLine('Third-party license notices for squirrel_translate')
    [void]$text.AppendLine('Go runtime')
    [void]$text.AppendLine([IO.File]::ReadAllText((Join-Path $goRoot 'LICENSE')))
    foreach ($module in $modules) {
        $parts = $module -split '\|', 2
        if ($parts[0] -eq 'github.com/simiooo/squirrel_translate') { continue }
        if ($parts.Count -ne 2 -or -not $parts[1]) { throw "Missing module directory: $($parts[0])" }
        $licenses = @(Get-ChildItem -LiteralPath $parts[1] -File | Where-Object { $_.Name -match '^(LICENSE|LICENCE|COPYING|NOTICE)(\..*)?$' })
        if ($licenses.Count -eq 0) { throw "Missing license: $($parts[0])" }
        [void]$text.AppendLine("`n===== $($parts[0]) =====")
        foreach ($license in $licenses) { [void]$text.AppendLine([IO.File]::ReadAllText($license.FullName)) }
    }
    [IO.File]::WriteAllText((Join-Path $root 'THIRD_PARTY_NOTICES.txt'), $text.ToString(), (New-Object Text.UTF8Encoding($false)))
} finally { Pop-Location }
