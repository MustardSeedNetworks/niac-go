# =============================================================================
# NiAC - Windows Distribution Build Script
# =============================================================================
# Creates a zip distribution with the binary and install helper.
#
# Usage:
#   .\build.ps1 [-Version "1.0.0"] [-Arch "amd64"]
#
# =============================================================================

param(
    [string]$Version = "",
    [string]$Arch = "amd64"
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent (Split-Path -Parent $ScriptDir)
$DistDir = Join-Path $RepoRoot "dist"
$BinaryName = "niac.exe"

# Auto-detect version from binary if not provided
if (-not $Version) {
    $BinaryPath = Join-Path $RepoRoot $BinaryName
    if (Test-Path $BinaryPath) {
        $VersionOutput = & $BinaryPath --version 2>$null
        $Version = [regex]::Match($VersionOutput, '\d+\.\d+\.\d+').Value
    }
    if (-not $Version) { $Version = "0.0.0" }
}

$PackageName = "niac-${Version}-windows-${Arch}"
$BuildDir = Join-Path $DistDir $PackageName

Write-Host "Building Windows distribution for NiAC" -ForegroundColor Green
Write-Host "  Version:      $Version"
Write-Host "  Architecture: $Arch"
Write-Host ""

# Clean and create build directory
Write-Host "[1/4] Preparing build directory..." -ForegroundColor Cyan
if (Test-Path $BuildDir) { Remove-Item -Recurse -Force $BuildDir }
New-Item -ItemType Directory -Force -Path $BuildDir | Out-Null

# Copy binary
Write-Host "[2/4] Copying binary..." -ForegroundColor Cyan
$SourceBinary = Join-Path $RepoRoot "niac-windows-${Arch}.exe"
if (-not (Test-Path $SourceBinary)) {
    $SourceBinary = Join-Path $RepoRoot $BinaryName
}
if (-not (Test-Path $SourceBinary)) {
    Write-Error "Cannot find niac binary. Download the Windows artifact from GitHub Releases or build on Windows first."
    exit 1
}
Copy-Item $SourceBinary (Join-Path $BuildDir $BinaryName)

# Copy install script
Write-Host "[3/4] Copying install script..." -ForegroundColor Cyan
$Installer = Join-Path $ScriptDir "install.ps1"
if (-not (Test-Path $Installer)) {
    Write-Error "Cannot find dedicated installer: $Installer"
    exit 1
}
Copy-Item $Installer (Join-Path $BuildDir "install.ps1")

# Create zip
Write-Host "[4/4] Creating zip archive..." -ForegroundColor Cyan
$ZipPath = Join-Path $DistDir "${PackageName}.zip"
if (Test-Path $ZipPath) { Remove-Item $ZipPath }
Compress-Archive -Path "$BuildDir\*" -DestinationPath $ZipPath

# Clean up build directory
Remove-Item -Recurse -Force $BuildDir

Write-Host ""
Write-Host "Distribution built successfully!" -ForegroundColor Green
Write-Host "  Output: $ZipPath"
Write-Host ""
Write-Host "  To install (as Administrator):"
Write-Host "    Expand-Archive $ZipPath -DestinationPath ."
Write-Host "    cd $PackageName"
Write-Host "    .\install.ps1"
