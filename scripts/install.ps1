#Requires -Version 5.1
<#
.SYNOPSIS
    nerv-task installer for Windows: downloads the right release,
    installs the nerv-task.exe binary, and runs `nerv-task install`.

.DESCRIPTION
    Usage:
      irm https://raw.githubusercontent.com/war-apps/nerv-task-by-gentle-ai/main/scripts/install.ps1 | iex

    Every option is an environment variable (this script takes no
    arguments of its own, since it is piped into iex):

      NERV_TASK_VERSION       Exact tag to install (e.g. "v1.2.0"). Default:
                               the latest stable release.
      NERV_TASK_INSTALL_DIR    Where to put the binary. Default:
                               $env:LOCALAPPDATA\nerv-task\bin.
      NERV_TASK_NO_INSTALL     When set to any non-empty value, skip
                               running `nerv-task install` after placing
                               the binary.
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Repo = "war-apps/nerv-task-by-gentle-ai"
$Binary = "nerv-task"
$ApiBase = "https://api.github.com/repos/$Repo"
$ReleaseBase = "https://github.com/$Repo/releases/download"

function Write-Log($Message) {
    Write-Host "nerv-task install.ps1: $Message"
}

function Get-Arch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    if ($env:PROCESSOR_ARCHITEW6432) {
        $arch = $env:PROCESSOR_ARCHITEW6432
    }
    switch -Regex ($arch) {
        "ARM64" { return "arm64" }
        "AMD64|x86_64" { return "amd64" }
        default { throw "unsupported architecture: $arch (see https://github.com/$Repo#install)" }
    }
}

function Resolve-Version {
    if ($env:NERV_TASK_VERSION) {
        return $env:NERV_TASK_VERSION
    }
    $release = Invoke-RestMethod -UseBasicParsing -Uri "$ApiBase/releases/latest"
    if (-not $release.tag_name) {
        throw "could not resolve the latest release from $ApiBase/releases/latest"
    }
    return $release.tag_name
}

function Install-NervTask {
    $goarch = Get-Arch
    $tag = Resolve-Version
    $version = $tag.TrimStart("v")
    $archiveName = "${Binary}_${version}_windows_${goarch}.zip"
    $baseUrl = "$ReleaseBase/$tag"

    $workDir = Join-Path $env:TEMP ("nerv-task-install-" + [System.Guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $workDir -Force | Out-Null
    try {
        $archivePath = Join-Path $workDir $archiveName
        $checksumsPath = Join-Path $workDir "checksums.txt"

        Write-Log "Downloading $archiveName ($tag)..."
        Invoke-WebRequest -UseBasicParsing -Uri "$baseUrl/$archiveName" -OutFile $archivePath
        Invoke-WebRequest -UseBasicParsing -Uri "$baseUrl/checksums.txt" -OutFile $checksumsPath

        Test-Checksum -ArchivePath $archivePath -ChecksumsPath $checksumsPath

        Expand-Archive -Path $archivePath -DestinationPath $workDir -Force
        $exePath = Join-Path $workDir "$Binary.exe"
        if (-not (Test-Path $exePath)) {
            throw "$Binary.exe was not found inside $archiveName"
        }

        $installDir = $env:NERV_TASK_INSTALL_DIR
        if (-not $installDir) {
            $installDir = Join-Path $env:LOCALAPPDATA "$Binary\bin"
        }
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
        $destExe = Join-Path $installDir "$Binary.exe"
        Copy-Item -Path $exePath -Destination $destExe -Force
        Write-Log "Installed $Binary to $destExe"

        Add-ToUserPath -Dir $installDir
        return $destExe
    }
    finally {
        Remove-Item -Path $workDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Test-Checksum {
    param([string]$ArchivePath, [string]$ChecksumsPath)

    $name = Split-Path -Leaf $ArchivePath
    $line = Select-String -Path $ChecksumsPath -Pattern ([Regex]::Escape($name)) -SimpleMatch | Select-Object -First 1
    if (-not $line) {
        throw "no checksum entry for $name in checksums.txt"
    }
    $expected = ($line.Line -split '\s+')[0]
    $actual = (Get-FileHash -Path $ArchivePath -Algorithm SHA256).Hash
    if ($expected.ToLower() -ne $actual.ToLower()) {
        throw "checksum mismatch for $name`: expected $expected, got $actual"
    }
}

function Add-ToUserPath {
    param([string]$Dir)

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if (-not $userPath) {
        $userPath = ""
    }
    $parts = $userPath -split ";" | Where-Object { $_ -ne "" }
    if ($parts -contains $Dir) {
        return
    }
    $newPath = if ($userPath) { "$userPath;$Dir" } else { $Dir }
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    $env:Path = "$env:Path;$Dir"
    Write-Log "Added $Dir to your user PATH. Restart your shell for it to take effect everywhere."
}

function Invoke-NervTaskInstall {
    param([string]$ExePath)

    if ($env:NERV_TASK_NO_INSTALL) {
        Write-Log "NERV_TASK_NO_INSTALL is set: skipping nerv-task install."
        return
    }
    & $ExePath install
}

$exe = Install-NervTask
Invoke-NervTaskInstall -ExePath $exe
