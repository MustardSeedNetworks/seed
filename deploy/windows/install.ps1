# Seed - Windows Installation Script
# Run as Administrator from an extracted Seed Windows archive.

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$InstallDir = "$env:ProgramFiles\Seed"
$BinaryName = "seed.exe"
$ServiceName = "SeedNetworkDiagnostics"
$SourceBinary = Join-Path $ScriptDir $BinaryName
$InstalledBinary = Join-Path $InstallDir $BinaryName

$Principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $Principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Error "Seed installation requires an elevated PowerShell session. Reopen PowerShell as Administrator and run install.ps1 again."
    exit 1
}

if (-not (Test-Path $SourceBinary)) {
    Write-Error "Cannot find $BinaryName next to install.ps1. Extract the complete archive before installing."
    exit 1
}

Write-Host "Installing Seed..." -ForegroundColor Green

$ExistingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($null -ne $ExistingService -and $ExistingService.Status -ne "Stopped") {
    Write-Host "  Stopping existing Windows service..."
    Stop-Service -Name $ServiceName -Force
    $ExistingService.WaitForStatus("Stopped", [TimeSpan]::FromSeconds(30))
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
New-Item -ItemType Directory -Force -Path "$InstallDir\configs" | Out-Null
New-Item -ItemType Directory -Force -Path "$InstallDir\logs" | Out-Null
New-Item -ItemType Directory -Force -Path "$InstallDir\data" | Out-Null
Copy-Item $SourceBinary $InstalledBinary -Force

$MachinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
if ($MachinePath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$MachinePath;$InstallDir", "Machine")
    Write-Host "  Added $InstallDir to system PATH"
}

Write-Host "  Installing Windows service..."
if ($null -eq $ExistingService) {
    & $InstalledBinary service install
    if ($LASTEXITCODE -ne 0) {
        $ExitCode = $LASTEXITCODE
        Write-Error "Seed service installation failed with exit code $ExitCode."
        exit $ExitCode
    }
} else {
    # Not sc.exe: Windows PowerShell 5.1 drops the embedded quotes around a
    # path with spaces when it builds a native command line.
    $Service = Get-CimInstance Win32_Service -Filter "Name='$ServiceName'"
    $Result = Invoke-CimMethod -InputObject $Service -MethodName Change `
        -Arguments @{ PathName = "`"$InstalledBinary`" service run" }
    if ($Result.ReturnValue -ne 0) {
        $ExitCode = $Result.ReturnValue
        Write-Error "Seed service reconfiguration failed with Win32_Service.Change result $ExitCode."
        exit $ExitCode
    }
}

Write-Host "  Adding firewall rule for port 8443..."
if ($null -eq (Get-NetFirewallRule -DisplayName "Seed" -ErrorAction SilentlyContinue)) {
    New-NetFirewallRule -DisplayName "Seed" `
        -Direction Inbound -Protocol TCP -LocalPort 8443 `
        -Action Allow | Out-Null
}

Write-Host "  Starting service..."
& $InstalledBinary service start
if ($LASTEXITCODE -ne 0) {
    $ExitCode = $LASTEXITCODE
    Write-Error "Seed service startup failed with exit code $ExitCode."
    exit $ExitCode
}

Write-Host ""
Write-Host "Installation complete!" -ForegroundColor Green
Write-Host "  Web UI: https://localhost:8443"
Write-Host "  Service: $ServiceName"
Write-Host ""
Write-Host "Commands:"
Write-Host "  seed service status   - Check service status"
Write-Host "  seed service stop     - Stop service"
Write-Host "  seed service start    - Start service"
