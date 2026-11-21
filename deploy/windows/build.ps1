# =============================================================================
# Seed - Windows Distribution Build Script
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
$BinaryName = "seed.exe"

# Auto-detect version from binary if not provided
if (-not $Version) {
    $BinaryPath = Join-Path $RepoRoot $BinaryName
    if (Test-Path $BinaryPath) {
        $Version = & $BinaryPath version 2>$null | Select-Object -First 1
        $Version = $Version -replace '^v', ''
    }
    if (-not $Version) { $Version = "0.0.0" }
}

$PackageName = "seed-${Version}-windows-${Arch}"
$BuildDir = Join-Path $DistDir $PackageName

Write-Host "Building Windows distribution for Seed" -ForegroundColor Green
Write-Host "  Version:      $Version"
Write-Host "  Architecture: $Arch"
Write-Host ""

# Clean and create build directory
Write-Host "[1/4] Preparing build directory..." -ForegroundColor Cyan
if (Test-Path $BuildDir) { Remove-Item -Recurse -Force $BuildDir }
New-Item -ItemType Directory -Force -Path $BuildDir | Out-Null

# Copy binary
Write-Host "[2/4] Copying binary..." -ForegroundColor Cyan
$SourceBinary = Join-Path $RepoRoot "seed-windows-${Arch}.exe"
if (-not (Test-Path $SourceBinary)) {
    $SourceBinary = Join-Path $RepoRoot $BinaryName
}
if (-not (Test-Path $SourceBinary)) {
    Write-Error "Cannot find seed binary. Build first with: make build-windows"
    exit 1
}
Copy-Item $SourceBinary (Join-Path $BuildDir $BinaryName)

# Create install script
Write-Host "[3/4] Creating install script..." -ForegroundColor Cyan
Copy-Item (Join-Path $ScriptDir "install.ps1") (Join-Path $BuildDir "install.ps1")

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
