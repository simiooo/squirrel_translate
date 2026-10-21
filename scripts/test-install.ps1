#requires -Version 5.1
param([Parameter(Mandatory=$true)][string]$Version,
      [ValidateSet('amd64','arm64')][string]$Architecture = 'amd64')
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$release = Join-Path $root "dist\release-$Architecture"
$sandbox = Join-Path ([IO.Path]::GetTempPath()) ('squirrel-test-' + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $sandbox | Out-Null
$install = Join-Path $sandbox 'app'
$argsForInstall = @{ Version=$Version; Architecture=$Architecture; InstallDirectory=$install; NoShortcut=$true;
    ArchivePath=(Join-Path $release "squirrel_translate-$Version-windows-$Architecture.zip");
    ChecksumsPath=(Join-Path $release "SHA256SUMS-$Architecture.txt") }
try {
    & "$PSScriptRoot\install.ps1" @argsForInstall
    $exe = Join-Path $install 'squirrel_translate.exe'
    if (-not (Test-Path -LiteralPath $exe)) { throw 'Install did not produce an executable.' }
    $originalHash = (Get-FileHash -LiteralPath $exe).Hash
    [IO.File]::WriteAllText((Join-Path $install 'user-sentinel.txt'), 'keep me')
    & "$PSScriptRoot\install.ps1" @argsForInstall
    if ((Get-FileHash -LiteralPath $exe).Hash -ne $originalHash) { throw 'Reinstall changed executable.' }
    $badSums = Join-Path $sandbox 'bad-sums.txt'
    [IO.File]::WriteAllText($badSums, (('0' * 64) + '  ' + [IO.Path]::GetFileName($argsForInstall.ArchivePath)))
    $bad = $argsForInstall.Clone(); $bad.ChecksumsPath = $badSums
    $rejected = $false
    try { & "$PSScriptRoot\install.ps1" @bad } catch { $rejected = $_.Exception.Message -match 'SHA256 verification failed' }
    if (-not $rejected -or (Get-FileHash -LiteralPath $exe).Hash -ne $originalHash) { throw 'Checksum failure did not preserve the installation.' }
    & "$PSScriptRoot\install.ps1" -Uninstall -InstallDirectory $install
    if (Test-Path -LiteralPath $exe) { throw 'Uninstall left the executable.' }
    if ([IO.File]::ReadAllText((Join-Path $install 'user-sentinel.txt')) -ne 'keep me') { throw 'Uninstall removed unrelated data.' }
    $rejected = $false
    try { & "$PSScriptRoot\install.ps1" @argsForInstall } catch { $rejected = $_.Exception.Message -match 'not managed' }
    if (-not $rejected) { throw 'Unowned directory was accepted.' }
    Write-Output 'PASS: install, reinstall, checksum rejection, uninstall, unrelated-file preservation and ownership checks.'
} finally {
    $tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\') + '\'
    $resolved = [IO.Path]::GetFullPath($sandbox)
    if ($resolved.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase) -and [IO.Path]::GetFileName($resolved) -match '^squirrel-test-[a-f0-9]{32}$') {
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}
