# =============================================================================
# The Seed - Windows Distribution Build Script
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

Write-Host "Building Windows distribution for The Seed" -ForegroundColor Green
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
$InstallScript = @'
# Seed - Windows Installation Script
# Run as Administrator

$ErrorActionPreference = "Stop"

$InstallDir = "$env:ProgramFiles\Seed"
$BinaryName = "seed.exe"
$ServiceName = "SeedNetworkDiagnostics"

Write-Host "Installing The Seed..." -ForegroundColor Green

# Create installation directory
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
New-Item -ItemType Directory -Force -Path "$InstallDir\configs" | Out-Null
New-Item -ItemType Directory -Force -Path "$InstallDir\logs" | Out-Null
New-Item -ItemType Directory -Force -Path "$InstallDir\data" | Out-Null

# Copy binary
Copy-Item $BinaryName "$InstallDir\$BinaryName" -Force

# Add to PATH
$MachinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
if ($MachinePath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$MachinePath;$InstallDir", "Machine")
    Write-Host "  Added $InstallDir to system PATH"
}

# Install as Windows service
Write-Host "  Installing Windows service..."
& "$InstallDir\$BinaryName" service install

# Add firewall rule
Write-Host "  Adding firewall rule for port 8443..."
New-NetFirewallRule -DisplayName "The Seed" `
    -Direction Inbound -Protocol TCP -LocalPort 8443 `
    -Action Allow -ErrorAction SilentlyContinue | Out-Null

# Start service
Write-Host "  Starting service..."
& "$InstallDir\$BinaryName" service start

Write-Host ""
Write-Host "Installation complete!" -ForegroundColor Green
Write-Host "  Web UI: https://localhost:8443"
Write-Host "  Service: $ServiceName"
Write-Host ""
Write-Host "Commands:"
Write-Host "  seed service status   - Check service status"
Write-Host "  seed service stop     - Stop service"
Write-Host "  seed service start    - Start service"
'@
Set-Content -Path (Join-Path $BuildDir "install.ps1") -Value $InstallScript

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
