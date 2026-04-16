#Requires -Version 5.1
[CmdletBinding()]
param(
    [string]$Version = '',
    [string]$Path = '',
    [switch]$Help
)

$ErrorActionPreference = 'Stop'
$ProgressPreference    = 'SilentlyContinue'

$Usage = @'
PowerShell installer for topo on Windows.
Downloads a release from the Arm artifactory server, installs it under the
current user, and ensures the install directory is on your PATH.

Usage:
  irm https://raw.githubusercontent.com/arm/topo/refs/heads/main/scripts/install.ps1 | iex

To pass options, invoke the downloaded script as a scriptblock:
  & ([scriptblock]::Create((irm https://raw.githubusercontent.com/arm/topo/refs/heads/main/scripts/install.ps1))) -Version v4.0.0 -Path C:\tools\topo

Options:
  -Version VERSION   Install a specific version (e.g. v4.0.0). Default: latest.
  -Path DIRECTORY    Install the binary into DIRECTORY instead of auto-detecting.
  -Help              Show this help message.
'@

$BaseUrl    = 'https://artifacts.tools.arm.com/topo'
$BinaryName = 'topo'
$ExeName    = 'topo.exe'

function Get-Architecture {
    # PROCESSOR_ARCHITEW6432 is set when a 32-bit process runs on 64-bit Windows.
    $arch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
    switch ($arch) {
        'AMD64' { 'amd64' }
        'ARM64' { 'arm64' }
        default { throw "Unsupported architecture: $arch" }
    }
}

function Resolve-Version {
    param([string]$Requested)

    if ([string]::IsNullOrWhiteSpace($Requested)) {
        Write-Host 'Resolving latest version...'
        $page  = (Invoke-WebRequest -UseBasicParsing -Uri "$BaseUrl/").Content
        $found = [regex]::Matches($page, 'v[0-9]+\.[0-9]+\.[0-9]+') |
                 ForEach-Object { $_.Value } |
                 Select-Object -Unique
        if (-not $found) { throw "Could not determine latest version from $BaseUrl/" }
        return ($found | Sort-Object { [version]($_.TrimStart('v')) } | Select-Object -Last 1)
    }

    if ($Requested -notmatch '^v') { return "v$Requested" }
    return $Requested
}

function Build-DownloadUrl {
    param([string]$Version)
    $arch    = Get-Architecture
    $archive = "${BinaryName}_windows_${arch}.zip"
    return "$BaseUrl/$Version/windows/$archive"
}

function Resolve-InstallDir {
    param([string]$Requested)

    if (-not [string]::IsNullOrWhiteSpace($Requested)) {
        New-Item -ItemType Directory -Path $Requested -Force | Out-Null
        return (Resolve-Path -LiteralPath $Requested).ProviderPath
    }

    $existing = Get-Command $ExeName -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($existing) {
        $dir = Split-Path -Parent $existing.Source
        $currentVersion = ''
        try { $currentVersion = (& $existing.Source --version 2>$null) -join ' ' } catch { }
        Write-Host "Existing installation found at $dir ($currentVersion), will update in-place"
        return $dir
    }

    $default = Join-Path $env:LOCALAPPDATA 'Programs\topo'
    New-Item -ItemType Directory -Path $default -Force | Out-Null
    return (Resolve-Path -LiteralPath $default).ProviderPath
}

function Install-Binary {
    param(
        [string]$Url,
        [string]$InstallDir,
        [string]$Version
    )

    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
    New-Item -ItemType Directory -Path $tmp -Force | Out-Null
    try {
        $archive = Join-Path $tmp ([System.IO.Path]::GetFileName($Url))
        Write-Host "Downloading $Url..."
        Invoke-WebRequest -UseBasicParsing -Uri $Url -OutFile $archive

        Expand-Archive -LiteralPath $archive -DestinationPath $tmp -Force

        $src = Get-ChildItem -Path $tmp -Recurse -Filter $ExeName | Select-Object -First 1
        if (-not $src) { throw "$ExeName not found in archive" }

        $dst = Join-Path $InstallDir $ExeName
        Copy-Item -LiteralPath $src.FullName -Destination $dst -Force
        Write-Host "Installed $BinaryName $Version to $dst"
    } finally {
        Remove-Item -Recurse -Force -LiteralPath $tmp -ErrorAction SilentlyContinue
    }
}

function Add-ToUserPath {
    param([string]$Dir)

    $userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
    if (-not $userPath) { $userPath = '' }
    $entries  = $userPath -split ';' | Where-Object { $_ }

    if ($entries -notcontains $Dir) {
        $newPath = if ($userPath) { "$userPath;$Dir" } else { $Dir }
        [Environment]::SetEnvironmentVariable('PATH', $newPath, 'User')
        Write-Host ""
        Write-Host "Added $Dir to your user PATH."
    } else {
        Write-Host ""
        Write-Host "Run '$BinaryName --help' to get started."
    }

    # Make the binary usable in the current session too.
    if (($env:PATH -split ';') -notcontains $Dir) {
        $env:PATH = "$env:PATH;$Dir"
    }
}

if ($Help) {
    Write-Host $Usage
    return
}

$resolvedVersion = Resolve-Version -Requested $Version
Write-Host "Installing $BinaryName $resolvedVersion"

$url        = Build-DownloadUrl -Version $resolvedVersion
$installDir = Resolve-InstallDir -Requested $Path

Install-Binary -Url $url -InstallDir $installDir -Version $resolvedVersion
Add-ToUserPath -Dir $installDir
