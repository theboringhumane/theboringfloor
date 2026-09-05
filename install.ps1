[CmdletBinding()]
param(
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA 'theboringfloor\bin')
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$App = 'theboringfloor'
$AssetApp = 'theboringoffice'
$Repo = 'theboringhumane/theboringoffice'
$ApiLatest = "https://api.github.com/repos/$Repo/releases/latest"

function Write-Stage([string]$Message) {
    Write-Host "==> $Message"
}

function Stop-Install([string]$Message) {
    throw "theboringoffice install failed: $Message"
}

function Get-WindowsArchitecture {
    $architecture = if ($env:PROCESSOR_ARCHITEW6432) {
        $env:PROCESSOR_ARCHITEW6432
    }
    else {
        $env:PROCESSOR_ARCHITECTURE
    }
    switch ($architecture) {
        'AMD64' { return 'amd64' }
        'ARM64' { return 'arm64' }
        default { Stop-Install "Windows $architecture is not supported. Releases ship amd64 and arm64 binaries." }
    }
}

function Get-ReleaseAsset($Release, [string]$Name) {
    $asset = @($Release.assets | Where-Object { $_.name -eq $Name })
    if ($asset.Count -ne 1) {
        Stop-Install "release $($Release.tag_name) does not contain $Name. See https://github.com/$Repo/releases for available downloads."
    }
    return $asset[0]
}

function Get-Checksum([string]$Path, [string]$AssetName) {
    $line = @(Get-Content -LiteralPath $Path | Where-Object {
        $_ -match ('^[a-fA-F0-9]{64}\s+\*?' + [regex]::Escape($AssetName) + '$')
    })
    if ($line.Count -ne 1) {
        Stop-Install "could not find one SHA-256 checksum for $AssetName."
    }
    return ($line[0] -split '\s+')[0].ToLowerInvariant()
}

if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT) {
    Stop-Install 'this installer is for Windows. On macOS or Linux, use install.sh instead.'
}
if ([string]::IsNullOrWhiteSpace($env:LOCALAPPDATA) -and -not $PSBoundParameters.ContainsKey('InstallDir')) {
    Stop-Install 'LOCALAPPDATA is unavailable. Re-run with -InstallDir <directory>.'
}

$architecture = Get-WindowsArchitecture
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("theboringoffice-install-" + [guid]::NewGuid())

try {
    Write-Stage "Detecting Windows architecture ($architecture)"
    Write-Stage 'Resolving the latest release'
    try {
        $release = Invoke-RestMethod -Uri $ApiLatest -Headers @{ 'User-Agent' = 'theboringoffice-installer' }
    }
    catch {
        Stop-Install "could not read the latest GitHub release: $($_.Exception.Message)"
    }

    $version = $release.tag_name -replace '^v', ''
    if ([string]::IsNullOrWhiteSpace($version)) {
        Stop-Install 'the latest GitHub release did not provide a version tag.'
    }

    $archiveName = "${AssetApp}_${version}_windows_${architecture}.zip"
    $checksumsName = "${AssetApp}_${version}_checksums.txt"
    $archive = Get-ReleaseAsset $release $archiveName
    $checksums = Get-ReleaseAsset $release $checksumsName

    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    $archivePath = Join-Path $tempDir $archiveName
    $checksumsPath = Join-Path $tempDir $checksumsName

    Write-Stage "Downloading $archiveName"
    try {
        Invoke-WebRequest -Uri $archive.browser_download_url -OutFile $archivePath
        Invoke-WebRequest -Uri $checksums.browser_download_url -OutFile $checksumsPath
    }
    catch {
        Stop-Install "download failed: $($_.Exception.Message)"
    }

    Write-Stage 'Verifying SHA-256 checksum'
    $expectedHash = Get-Checksum $checksumsPath $archiveName
    $actualHash = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualHash -ne $expectedHash) {
        Stop-Install "checksum mismatch for $archiveName; the download was not installed."
    }

    $extractDir = Join-Path $tempDir 'extract'
    Write-Stage 'Extracting the release archive'
    Expand-Archive -LiteralPath $archivePath -DestinationPath $extractDir -Force
    $primaryBinaries = @(Get-ChildItem -LiteralPath $extractDir -Recurse -File -Filter "${App}.exe")
    if ($primaryBinaries.Count -ne 1) {
        $primaryBinaries = @(Get-ChildItem -LiteralPath $extractDir -Recurse -File -Filter "${AssetApp}.exe")
    }
    if ($primaryBinaries.Count -ne 1) {
        Stop-Install "expected one $App.exe or $AssetApp.exe in $archiveName, found $($primaryBinaries.Count)."
    }
    $mcpBinaries = @(Get-ChildItem -LiteralPath $extractDir -Recurse -File -Filter 'thefloor_mcp.exe')
    if ($mcpBinaries.Count -gt 1) {
        Stop-Install "expected at most one thefloor_mcp.exe in $archiveName, found $($mcpBinaries.Count)."
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $destination = Join-Path $InstallDir "${App}.exe"
    $mcpDestination = Join-Path $InstallDir 'thefloor_mcp.exe'
    Write-Stage "Installing ${App}.exe to $InstallDir"
    try {
        Copy-Item -LiteralPath $primaryBinaries[0].FullName -Destination $destination -Force
        Copy-Item -LiteralPath $destination -Destination (Join-Path $InstallDir 'tbo.exe') -Force
        if ($mcpBinaries.Count -eq 1) {
            Copy-Item -LiteralPath $mcpBinaries[0].FullName -Destination $mcpDestination -Force
            Write-Host "Installed thefloor_mcp.exe to $InstallDir"
        }
        else {
            Write-Host 'thefloor_mcp.exe was not included in this archive; continuing with theboringfloor.exe only.'
        }
    }
    catch {
        Stop-Install "could not replace $destination. Close a running theboringoffice process and try again. $($_.Exception.Message)"
    }

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $pathEntries = @($userPath -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    $onPath = $pathEntries | Where-Object { $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\') }
    if (-not $onPath) {
        [Environment]::SetEnvironmentVariable('Path', (($pathEntries + $InstallDir) -join ';'), 'User')
        $env:Path = "$InstallDir;$env:Path"
        Write-Host "Added $InstallDir to your user PATH. Open a new PowerShell window before your next session."
    }
    else {
        Write-Host "$InstallDir is already on your user PATH."
    }

    Write-Host ''
    Write-Host 'theboringfloor installed successfully.'
    Write-Host 'Run: theboringfloor --demo   (or tbo --demo)'
    Write-Host 'Companion: thefloor_mcp'
}
finally {
    if (Test-Path -LiteralPath $tempDir) {
        Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
