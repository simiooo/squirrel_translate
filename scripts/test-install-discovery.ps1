#requires -Version 5.1
param([string]$Version = '0.1.0', [ValidateSet('amd64','arm64')][string]$Architecture = 'amd64')
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$root = Split-Path -Parent $PSScriptRoot
$sandbox = Join-Path ([IO.Path]::GetTempPath()) ('squirrel-test-' + [Guid]::NewGuid().ToString('N'))
$key = 'HKCU:\Software\SquirrelTranslateInstallerTests\' + [Guid]::NewGuid().ToString('N')
New-Item -ItemType Directory -Path $sandbox | Out-Null
function Assert($condition, $message) { if (-not $condition) { throw $message } }
function Hash($path) { (Get-FileHash -LiteralPath $path).Hash }
try {
    # Redirect only discovery roots; never touch the user's install or shortcuts.
    $script = Get-Content -LiteralPath "$PSScriptRoot\install.ps1" -Raw
    $script = $script.Replace("[Environment]::GetFolderPath('Programs')", "'$sandbox'")
    $script = $script.Replace("Join-Path `$env:LOCALAPPDATA 'Programs\squirrel_translate-portable'", "Join-Path '$sandbox' 'default'")
    $script = $script.Replace('HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\io.github.simiooo.squirrel_translate', $key)
    $script = $script.Replace('HKCU:\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\io.github.simiooo.squirrel_translate', ($key + '-32'))
    $runner = Join-Path $sandbox 'install.ps1'
    [IO.File]::WriteAllText($runner, $script, (New-Object Text.UTF8Encoding($true)))
    $argsInstall = @{Version=$Version; Architecture=$Architecture; ArchivePath=(Join-Path $root "dist\release-$Architecture\squirrel_translate-$Version-windows-$Architecture.zip"); ChecksumsPath=(Join-Path $root "dist\release-$Architecture\SHA256SUMS-$Architecture.txt")}
    $custom = Join-Path $sandbox 'custom path'
    & $runner @argsInstall -InstallDirectory $custom
    $exe = Join-Path $custom 'squirrel_translate.exe'
    $expected = Hash $exe
    $shortcut = Join-Path $sandbox 'Squirrel Translate.lnk'
    $link = (New-Object -ComObject WScript.Shell).CreateShortcut($shortcut)
    $link.Arguments = '--tray --data-dir "D:\custom data"'
    $link.Save()
    $shortcutHash = Hash $shortcut
    [IO.File]::WriteAllText((Join-Path $custom 'settings.json'), 'keep configuration')
    [IO.File]::WriteAllText($exe, 'old custom executable')
    & $runner @argsInstall
    Assert ((Hash $exe) -eq $expected) 'Custom path discovery failed'
    Assert ((Hash $shortcut) -eq $shortcutHash) 'Shortcut changed'
    Assert ([IO.File]::ReadAllText((Join-Path $custom 'settings.json')) -eq 'keep configuration') 'Config changed'
    $nsis = Join-Path $sandbox 'installer path'
    New-Item -ItemType Directory -Path $nsis | Out-Null
    $nsisExe = Join-Path $nsis 'Squirrel Translate.exe'
    [IO.File]::WriteAllText($nsisExe, 'old installer executable')
    [IO.File]::WriteAllText((Join-Path $nsis 'Uninstall.exe'), 'keep uninstaller')
    New-Item -Path $key -Force | Out-Null
    New-ItemProperty -LiteralPath $key -Name InstallLocation -Value $nsis | Out-Null
    New-ItemProperty -LiteralPath $key -Name DisplayIcon -Value $nsisExe | Out-Null
    New-ItemProperty -LiteralPath $key -Name DisplayVersion -Value '0.0.1' | Out-Null
    $rejected = $false
    try { & $runner @argsInstall } catch { $rejected = $_ -match 'Multiple installations' }
    Assert $rejected 'Ambiguous discovery accepted'
    & $runner @argsInstall -InstallDirectory $nsis -NoShortcut
    Assert ((Hash $nsisExe) -eq $expected) 'Installer executable not replaced'
    Assert (-not (Test-Path (Join-Path $nsis 'squirrel_translate.exe'))) 'Wrong executable name'
    Assert ((Get-ItemProperty -LiteralPath $key).DisplayVersion -eq $Version) 'Registry version not updated'
    Assert ([IO.File]::ReadAllText((Join-Path $nsis 'Uninstall.exe')) -eq 'keep uninstaller') 'Uninstaller changed'
    Assert ((Hash $shortcut) -eq $shortcutHash) 'Other shortcut changed'
    Remove-Item -LiteralPath $shortcut
    & $runner @argsInstall -NoShortcut
    Assert (-not (Test-Path (Join-Path $sandbox 'default'))) 'Registry discovery missed'
    [IO.File]::WriteAllText($nsisExe, 'old before failed upgrade')
    $oldHash = Hash $nsisExe
    Set-ItemProperty -LiteralPath $key -Name DisplayVersion -Value '0.0.2'
    $lock = [IO.File]::Open((Join-Path $nsis 'LICENSE'), 'Open', 'Read', 'Read')
    $failed = $false
    try { & $runner @argsInstall -NoShortcut } catch { $failed = $true } finally { $lock.Dispose() }
    Assert $failed 'Locked file did not fail'
    Assert ((Hash $nsisExe) -eq $oldHash) 'Partial replacement did not restore executable'
    Assert ((Get-ItemProperty -LiteralPath $key).DisplayVersion -eq '0.0.2') 'Failed update changed registration'
    $broken = $script.Replace('Set-ItemProperty -LiteralPath $registration.Key -Name DisplayVersion -Value $Version', "throw 'Simulated metadata failure'")
    [IO.File]::WriteAllText($runner, $broken, (New-Object Text.UTF8Encoding($true)))
    $failed = $false
    try { & $runner @argsInstall } catch { $failed = $_ -match 'Simulated metadata failure' }
    Assert $failed 'Metadata failure not injected'
    Assert ((Hash $nsisExe) -eq $oldHash) 'Metadata failure did not restore executable'
    Assert (-not (Test-Path $shortcut)) 'Failed update left a shortcut'
    Assert ((Get-ItemProperty -LiteralPath $key).DisplayVersion -eq '0.0.2') 'Metadata rollback failed'
    # Restore the unmodified script and identify a ZIP install by executable metadata.
    [IO.File]::WriteAllText($runner, $script, (New-Object Text.UTF8Encoding($true)))
    Remove-Item -LiteralPath $key
    $portable = Join-Path $sandbox 'portable path'
    New-Item -ItemType Directory -Path $portable | Out-Null
    $portableExe = Join-Path $portable 'squirrel_translate.exe'
    Copy-Item -LiteralPath $exe -Destination $portableExe
    [IO.File]::WriteAllText((Join-Path $portable 'user-settings.json'), 'keep portable data')
    $link = (New-Object -ComObject WScript.Shell).CreateShortcut($shortcut)
    $link.TargetPath = $portableExe
    $link.Arguments = '--tray'
    $link.Save()
    $portableShortcutHash = Hash $shortcut
    & $runner @argsInstall
    Assert ((Hash $portableExe) -eq $expected) 'Portable discovery failed'
    Assert (-not (Test-Path (Join-Path $portable '.squirrel-install.json'))) 'Portable install converted to script install'
    Assert ((Hash $shortcut) -eq $portableShortcutHash) 'Portable shortcut changed'
    Assert ([IO.File]::ReadAllText((Join-Path $portable 'user-settings.json')) -eq 'keep portable data') 'Portable data changed'
    Remove-Item -LiteralPath $shortcut
    & $runner @argsInstall -InstallDirectory $portable -NoShortcut
    Write-Output 'PASS: custom, installer and portable discovery; ambiguity; explicit selection; shortcut/config/uninstaller preservation; locked-file and metadata rollback.'
} finally {
    # Remove only this test's exact registry key and individual sandbox files.
    if ($key -match '^HKCU:\\Software\\SquirrelTranslateInstallerTests\\[a-f0-9]{32}$') { Remove-Item -LiteralPath $key -Force -ErrorAction SilentlyContinue }
    $resolved = [IO.Path]::GetFullPath($sandbox)
    $tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\') + '\'
    if ($resolved.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase) -and [IO.Path]::GetFileName($resolved) -match '^squirrel-test-[a-f0-9]{32}$') {
        Get-ChildItem -LiteralPath $resolved -Recurse -File -Force | ForEach-Object { Remove-Item -LiteralPath $_.FullName -Force }
        Get-ChildItem -LiteralPath $resolved -Recurse -Directory | Sort-Object { $_.FullName.Length } -Descending | ForEach-Object { Remove-Item -LiteralPath $_.FullName }
        Remove-Item -LiteralPath $resolved
    }
}
