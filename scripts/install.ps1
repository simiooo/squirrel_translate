#requires -Version 5.1
[CmdletBinding()]
param(
    [ValidatePattern('^(latest|v?\d+\.\d+\.\d+)$')][string]$Version = 'latest',
    [ValidateSet('auto','amd64','arm64')][string]$Architecture = 'auto',
    [string]$InstallDirectory = (Join-Path $env:LOCALAPPDATA 'Programs\squirrel_translate-portable'),
    [switch]$Uninstall,
    [switch]$NoShortcut,
    # Offline/testing mode still requires the published SHA256SUMS file.
    [string]$ArchivePath,
    [string]$ChecksumsPath
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$repo = 'simiooo/squirrel_translate'
$destination = [IO.Path]::GetFullPath($InstallDirectory).TrimEnd('\')
$marker = Join-Path $destination '.squirrel-install.json'
$exe = Join-Path $destination 'squirrel_translate.exe'
$shortcut = Join-Path ([Environment]::GetFolderPath('Programs')) 'Squirrel Translate.lnk'

function Assert-OwnedDirectory {
    if (-not (Test-Path -LiteralPath $marker -PathType Leaf)) { throw 'This directory is not managed by the squirrel_translate script installer.' }
    $record = Get-Content -LiteralPath $marker -Raw | ConvertFrom-Json
    if ($record.repository -ne $repo -or $record.directory -ne $destination) { throw 'Installation ownership check failed.' }
    return $record
}
function Assert-NotRunning {
    $running = @(Get-Process -Name 'squirrel_translate' -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $exe })
    if ($running.Count) { throw 'Quit Squirrel Translate from the tray before installing or uninstalling.' }
}
if ($destination -eq [IO.Path]::GetPathRoot($destination).TrimEnd('\') -or $destination -eq [IO.Path]::GetFullPath($env:USERPROFILE).TrimEnd('\')) {
    throw 'Choose a dedicated application directory.'
}
if ($Uninstall) {
    $record = Assert-OwnedDirectory
    Assert-NotRunning
    # Never recursively delete a user-selected directory; remove only our fixed files.
    foreach ($name in @('squirrel_translate.exe','LICENSE','THIRD_PARTY_NOTICES.txt','PRIVACY.md','.squirrel-install.json')) {
        $path = Join-Path $destination $name
        if (Test-Path -LiteralPath $path) { Remove-Item -LiteralPath $path -Force }
    }
    if ($record.shortcut -and (Test-Path -LiteralPath $shortcut)) {
        $link = (New-Object -ComObject WScript.Shell).CreateShortcut($shortcut)
        if ($link.TargetPath -eq $exe) { Remove-Item -LiteralPath $shortcut -Force }
    }
    if (@(Get-ChildItem -LiteralPath $destination -Force).Count -eq 0) { Remove-Item -LiteralPath $destination }
    Write-Output 'Uninstalled. Settings, vocabulary and credentials were preserved.'
    return
}
if ($Architecture -eq 'auto') {
    $nativeArch = $env:PROCESSOR_ARCHITECTURE
    if ($env:PROCESSOR_ARCHITEW6432) { $nativeArch = $env:PROCESSOR_ARCHITEW6432 }
    if ($nativeArch -eq 'ARM64') { $Architecture = 'arm64' }
    elseif ($nativeArch -eq 'AMD64') { $Architecture = 'amd64' }
    else { throw 'Only Windows x64 and ARM64 are supported.' }
}
if (Test-Path -LiteralPath $destination) {
    if (@(Get-ChildItem -LiteralPath $destination -Force).Count -gt 0) { $null = Assert-OwnedDirectory }
}
if (Test-Path -LiteralPath $shortcut) {
    $existingLink = (New-Object -ComObject WScript.Shell).CreateShortcut($shortcut)
    if (-not $NoShortcut -and $existingLink.TargetPath -ne $exe) { throw 'A different installation owns the Start menu shortcut. Uninstall it first or use -NoShortcut.' }
}
Assert-NotRunning
$temp = Join-Path ([IO.Path]::GetTempPath()) ('squirrel-install-' + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $temp | Out-Null
try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
    if (-not $ArchivePath) {
        $headers = @{ 'User-Agent' = 'squirrel_translate-installer'; Accept = 'application/vnd.github+json' }
        if ($Version -eq 'latest') { $api = "https://api.github.com/repos/$repo/releases/latest" }
        else { $api = "https://api.github.com/repos/$repo/releases/tags/v$($Version.TrimStart('v'))" }
        $release = Invoke-RestMethod -Uri $api -Headers $headers
        if ($release.tag_name -notmatch '^v\d+\.\d+\.\d+$' -or $release.draft -or $release.prerelease) { throw 'Expected a published stable version.' }
        $Version = $release.tag_name.Substring(1)
        $name = "squirrel_translate-$Version-windows-$Architecture.zip"
        $sumName = "SHA256SUMS-$Architecture.txt"
        foreach ($assetName in @($name, $sumName)) {
            $assets = @($release.assets | Where-Object { $_.name -ceq $assetName })
            if ($assets.Count -ne 1) { throw "Missing or ambiguous release asset: $assetName" }
            $url = $assets[0].browser_download_url
            if (-not $url.StartsWith("https://github.com/$repo/releases/download/v$Version/", [StringComparison]::Ordinal)) { throw 'Unexpected download origin.' }
            Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile (Join-Path $temp $assetName)
        }
        $ArchivePath = Join-Path $temp $name
        $ChecksumsPath = Join-Path $temp $sumName
    } elseif (-not $ChecksumsPath -or $Version -eq 'latest') { throw 'Offline install requires -Version and -ChecksumsPath.' }
    $Version = $Version.TrimStart('v')
    $name = "squirrel_translate-$Version-windows-$Architecture.zip"
    if ([IO.Path]::GetFileName($ArchivePath) -cne $name) { throw 'Archive name does not match version and architecture.' }
    $pattern = '^([a-fA-F0-9]{64})  ' + [regex]::Escape($name) + '$'
    $matchesFound = @(Get-Content -LiteralPath $ChecksumsPath | Where-Object { $_ -cmatch $pattern })
    if ($matchesFound.Count -ne 1) { throw 'Expected exactly one matching checksum.' }
    $expected = [regex]::Match($matchesFound[0], $pattern).Groups[1].Value
    if ((Get-FileHash -LiteralPath $ArchivePath -Algorithm SHA256).Hash -ne $expected) { throw 'SHA256 verification failed. Nothing was installed.' }
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $archive = [IO.Compression.ZipFile]::OpenRead((Resolve-Path -LiteralPath $ArchivePath).Path)
    $allowed = @('squirrel_translate.exe','LICENSE','THIRD_PARTY_NOTICES.txt','PRIVACY.md')
    try {
        if ($archive.Entries.Count -ne $allowed.Count) { throw 'Unexpected archive contents.' }
        foreach ($file in $allowed) {
            $entries = @($archive.Entries | Where-Object { $_.FullName -ceq $file })
            if ($entries.Count -ne 1 -or $entries[0].Length -gt 256MB) { throw "Invalid archive entry: $file" }
            [IO.Compression.ZipFileExtensions]::ExtractToFile($entries[0], (Join-Path $temp $file), $false)
        }
    } finally { $archive.Dispose() }
    # Stage and validate everything before changing the installation.
    New-Item -ItemType Directory -Path $destination -Force | Out-Null
    $backup = Join-Path $temp 'backup'
    New-Item -ItemType Directory -Path $backup | Out-Null
    $changed = @()
    try {
        foreach ($file in ($allowed + @('.squirrel-install.json'))) {
            $target = Join-Path $destination $file
            if (Test-Path -LiteralPath $target) { Copy-Item -LiteralPath $target -Destination (Join-Path $backup $file) }
        }
        foreach ($file in $allowed) {
            $changed += $file
            Copy-Item -LiteralPath (Join-Path $temp $file) -Destination (Join-Path $destination $file) -Force
        }
        if (-not $NoShortcut) {
            $link = (New-Object -ComObject WScript.Shell).CreateShortcut($shortcut)
            $link.TargetPath = $exe
            $link.WorkingDirectory = $destination
            $link.IconLocation = "$exe,0"
            $link.Description = 'Squirrel Translate'
            $link.Save()
        }
        $ownsShortcut = -not $NoShortcut
        if (Test-Path -LiteralPath $shortcut) { $ownsShortcut = ((New-Object -ComObject WScript.Shell).CreateShortcut($shortcut).TargetPath -eq $exe) }
        $record = @{repository=$repo;directory=$destination;version=$Version;architecture=$Architecture;shortcut=$ownsShortcut}
        $changed += '.squirrel-install.json'
        [IO.File]::WriteAllText($marker, ($record | ConvertTo-Json), (New-Object Text.UTF8Encoding($false)))
    } catch {
        foreach ($file in $changed) {
            $saved = Join-Path $backup $file
            $target = Join-Path $destination $file
            if (Test-Path -LiteralPath $saved) { Copy-Item -LiteralPath $saved -Destination $target -Force }
            elseif (Test-Path -LiteralPath $target) { Remove-Item -LiteralPath $target -Force }
        }
        throw
    }
    Write-Output "Installed Squirrel Translate $Version ($Architecture) to $destination"
} finally {
    # The only recursive deletion is our freshly allocated, validated temporary directory.
    $tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\') + '\'
    $resolvedTemp = [IO.Path]::GetFullPath($temp)
    if ($resolvedTemp.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase) -and [IO.Path]::GetFileName($resolvedTemp) -match '^squirrel-install-[a-f0-9]{32}$') {
        Remove-Item -LiteralPath $resolvedTemp -Recurse -Force
    }
}
