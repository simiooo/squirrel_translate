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
$defaultDirectory = Join-Path $env:LOCALAPPDATA 'Programs\squirrel_translate-portable'
$shortcut = Join-Path ([Environment]::GetFolderPath('Programs')) 'Squirrel Translate.lnk'
$registrationPaths = @(
    'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\io.github.simiooo.squirrel_translate',
    'HKCU:\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\io.github.simiooo.squirrel_translate'
)
$registrations = @()
foreach ($key in $registrationPaths) {
    if (Test-Path -LiteralPath $key) {
        $r = Get-ItemProperty -LiteralPath $key
        if ($r.PSObject.Properties['InstallLocation'] -and $r.PSObject.Properties['DisplayIcon']) {
            $dir = [IO.Path]::GetFullPath($r.InstallLocation).TrimEnd('\')
            $target = Join-Path $dir 'Squirrel Translate.exe'
            if ($r.DisplayIcon -eq $target -and (Test-Path -LiteralPath $target -PathType Leaf) -and
                (Test-Path -LiteralPath (Join-Path $dir 'Uninstall.exe') -PathType Leaf)) {
                $registrations += [pscustomobject]@{ Directory=$dir; Key=$key; Record=$r }
            }
        }
    }
}
function Test-ScriptInstall([string]$directory) {
    $path = Join-Path $directory '.squirrel-install.json'
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { return $false }
    try {
        $r = Get-Content -LiteralPath $path -Raw | ConvertFrom-Json
        return ($r.repository -eq $repo -and $r.directory -eq $directory)
    } catch { return $false }
}
function Test-PortableInstall([string]$directory) {
    $path = Join-Path $directory 'squirrel_translate.exe'
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { return $false }
    $product = (Get-Item -LiteralPath $path).VersionInfo.ProductName
    return ($product -in @('Squirrel Translate', 'squirrel_translate', [string]([char]0x677E) + [char]0x9F20 + [char]0x7FFB + [char]0x8BD1))
}
$existingLink = $null
if (Test-Path -LiteralPath $shortcut) { $existingLink = (New-Object -ComObject WScript.Shell).CreateShortcut($shortcut) }
if (-not $PSBoundParameters.ContainsKey('InstallDirectory')) {
    $candidates = @($registrations | ForEach-Object { $_.Directory })
    if ((Test-ScriptInstall $defaultDirectory) -or (Test-PortableInstall $defaultDirectory)) { $candidates += $defaultDirectory }
    if ($existingLink -and $existingLink.TargetPath) {
        $linkedDirectory = Split-Path -Parent $existingLink.TargetPath
        if ((Test-ScriptInstall $linkedDirectory) -or (Test-PortableInstall $linkedDirectory)) { $candidates += $linkedDirectory }
    }
    $candidates = @($candidates | Sort-Object -Unique)
    if ($candidates.Count -gt 1) { throw ('Multiple installations found. Choose -InstallDirectory: ' + ($candidates -join '; ')) }
    if ($candidates.Count -eq 1) { $InstallDirectory = $candidates[0] }
}
$destination = [IO.Path]::GetFullPath($InstallDirectory).TrimEnd('\')
$marker = Join-Path $destination '.squirrel-install.json'
$matchingRegistrations = @($registrations | Where-Object { $_.Directory -eq $destination })
$installerMode = $matchingRegistrations.Count -gt 0
$portableMode = -not $installerMode -and -not (Test-Path -LiteralPath $marker) -and (Test-PortableInstall $destination)
$exeName = 'squirrel_translate.exe'
if ($installerMode) { $exeName = 'Squirrel Translate.exe' }
$exe = Join-Path $destination $exeName
# Refuse links/junctions in the target path; ownership must refer to this directory.
$ancestor = $destination
while ($ancestor) {
    if ((Test-Path -LiteralPath $ancestor) -and ((Get-Item -LiteralPath $ancestor -Force).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Installation path must not contain reparse points.' }
    $ancestor = Split-Path -Parent $ancestor
}

function Assert-OwnedDirectory {
    if (-not (Test-Path -LiteralPath $marker -PathType Leaf)) { throw 'This directory is not managed by the squirrel_translate script installer.' }
    $record = Get-Content -LiteralPath $marker -Raw | ConvertFrom-Json
    if ($record.repository -ne $repo -or $record.directory -ne $destination) { throw 'Installation ownership check failed.' }
    return $record
}
function Assert-NotRunning {
    $running = @(Get-Process -Name ([IO.Path]::GetFileNameWithoutExtension($exe)) -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $exe })
    if ($running.Count) { throw 'Quit Squirrel Translate from the tray before installing or uninstalling.' }
}
if ($destination -eq [IO.Path]::GetPathRoot($destination).TrimEnd('\') -or $destination -eq [IO.Path]::GetFullPath($env:USERPROFILE).TrimEnd('\')) {
    throw 'Choose a dedicated application directory.'
}
if ($Uninstall) {
    if ($installerMode) { throw 'This installation uses the Windows installer. Uninstall it from Windows Settings.' }
    if ($portableMode) { throw 'This is a portable ZIP installation. Remove its application files to uninstall; keep your data directory.' }
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
    if (@(Get-ChildItem -LiteralPath $destination -Force).Count -gt 0 -and -not $installerMode -and -not $portableMode) { $null = Assert-OwnedDirectory }
}
if (Test-Path -LiteralPath $shortcut) {
    $existingLink = (New-Object -ComObject WScript.Shell).CreateShortcut($shortcut)
    if (-not $NoShortcut -and $existingLink.TargetPath -ne $exe) { throw 'A different installation owns the Start menu shortcut. Uninstall it first or use -NoShortcut.' }
}
Assert-NotRunning
$keepBackup = $false
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
    Assert-NotRunning
    # Stage and validate everything before changing the installation.
    New-Item -ItemType Directory -Path $destination -Force | Out-Null
    $backup = Join-Path $temp 'backup'
    New-Item -ItemType Directory -Path $backup | Out-Null
    $changed = @()
    $registrationChanged = @()
    $createdShortcut = $false
    $keepBackup = $false
    $targets = @($exeName, 'LICENSE', 'THIRD_PARTY_NOTICES.txt', 'PRIVACY.md')
    if (-not $installerMode -and -not $portableMode) { $targets += '.squirrel-install.json' }
    try {
        foreach ($file in $targets) {
            $target = Join-Path $destination $file
            if (Test-Path -LiteralPath $target) {
                $item = Get-Item -LiteralPath $target -Force
                if ($item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw "Not a regular installation file: $target" }
                Copy-Item -LiteralPath $target -Destination (Join-Path $backup $file)
            }
        }
        foreach ($file in $allowed) {
            $targetName = $file
            if ($file -eq 'squirrel_translate.exe') { $targetName = $exeName }
            # Rename the existing file first: a locked file must never be truncated.
            $target = Join-Path $destination $targetName
            $replacement = Join-Path $destination ('.squirrel-replace-' + [Guid]::NewGuid().ToString('N'))
            Copy-Item -LiteralPath (Join-Path $temp $file) -Destination $replacement
            try {
                if (Test-Path -LiteralPath $target) {
                    $old = Join-Path $destination ('.squirrel-old-' + [Guid]::NewGuid().ToString('N'))
                    Move-Item -LiteralPath $target -Destination $old
                    $changed += $targetName
                    try { Move-Item -LiteralPath $replacement -Destination $target }
                    finally { Remove-Item -LiteralPath $old -Force }
                } else {
                    $changed += $targetName
                    Move-Item -LiteralPath $replacement -Destination $target
                }
            } finally {
                if (Test-Path -LiteralPath $replacement) { Remove-Item -LiteralPath $replacement -Force }
            }
        }
        # Preserve existing shortcut bytes, including custom arguments and working directory.
        if (-not $NoShortcut -and -not (Test-Path -LiteralPath $shortcut)) {
            $link = (New-Object -ComObject WScript.Shell).CreateShortcut($shortcut)
            $link.TargetPath = $exe
            $link.WorkingDirectory = $destination
            $link.IconLocation = "$exe,0"
            $link.Description = 'Squirrel Translate'
            $createdShortcut = $true
            $link.Save()
        }
        if ($installerMode) {
            foreach ($registration in $matchingRegistrations) {
                $registrationChanged += $registration
                Set-ItemProperty -LiteralPath $registration.Key -Name DisplayVersion -Value $Version
            }
        } elseif (-not $portableMode) {
            $ownsShortcut = $false
            if (Test-Path -LiteralPath $shortcut) { $ownsShortcut = ((New-Object -ComObject WScript.Shell).CreateShortcut($shortcut).TargetPath -eq $exe) }
            $record = @{repository=$repo;directory=$destination;version=$Version;architecture=$Architecture;shortcut=$ownsShortcut}
            if (Test-Path -LiteralPath $marker) {
                $previous = Assert-OwnedDirectory
                foreach ($property in $previous.PSObject.Properties) {
                    if (-not $record.ContainsKey($property.Name)) { $record[$property.Name] = $property.Value }
                }
            }
            $changed += '.squirrel-install.json'
            [IO.File]::WriteAllText($marker, ($record | ConvertTo-Json -Depth 20), (New-Object Text.UTF8Encoding($false)))
        }
    } catch {
        $failure = $_
        foreach ($registration in $registrationChanged) {
            try {
                if ($registration.Record.PSObject.Properties['DisplayVersion']) {
                    Set-ItemProperty -LiteralPath $registration.Key -Name DisplayVersion -Value $registration.Record.DisplayVersion
                } else { Remove-ItemProperty -LiteralPath $registration.Key -Name DisplayVersion }
            } catch { $keepBackup = $true; Write-Warning $_ }
        }
        if ($createdShortcut) {
            try { Remove-Item -LiteralPath $shortcut -Force -ErrorAction Stop } catch { $keepBackup = $true; Write-Warning $_ }
        }
        [array]::Reverse($changed)
        foreach ($file in $changed) {
            try {
                $saved = Join-Path $backup $file
                $target = Join-Path $destination $file
                if (Test-Path -LiteralPath $saved) { Copy-Item -LiteralPath $saved -Destination $target -Force }
                elseif (Test-Path -LiteralPath $target) { Remove-Item -LiteralPath $target -Force }
            } catch { $keepBackup = $true; Write-Warning $_ }
        }
        if ($keepBackup) { throw "Upgrade failed and recovery needs attention. Backups preserved at $backup. Original error: $failure" }
        throw $failure
    }
    Write-Output "Installed Squirrel Translate $Version ($Architecture) to $destination"
} finally {
    # The only recursive deletion is our freshly allocated, validated temporary directory.
    $tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\') + '\'
    $resolvedTemp = [IO.Path]::GetFullPath($temp)
    if (-not $keepBackup -and $resolvedTemp.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase) -and [IO.Path]::GetFileName($resolvedTemp) -match '^squirrel-install-[a-f0-9]{32}$') {
        Remove-Item -LiteralPath $resolvedTemp -Recurse -Force
    }
}
