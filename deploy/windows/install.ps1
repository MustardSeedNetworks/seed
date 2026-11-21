# Seed - Windows Installation Script
# Run as Administrator from an extracted Seed Windows archive.

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$InstallDir = "$env:ProgramFiles\Seed"
$BinaryName = "seed.exe"
$ServiceName = "SeedNetworkDiagnostics"
$SourceBinary = Join-Path $ScriptDir $BinaryName
$InstalledBinary = Join-Path $InstallDir $BinaryName
# Everything the service writes (config, database, logs, certificates) lives
# under here. The Known Folder API is used rather than $env:ProgramData, which
# is whatever the launching process says it is; the service resolves the same
# folder the same way.
$DataDir = Join-Path ([Environment]::GetFolderPath("CommonApplicationData")) "Seed"
$AdministratorsSid = New-Object Security.Principal.SecurityIdentifier("S-1-5-32-544")
$SystemSid = New-Object Security.Principal.SecurityIdentifier("S-1-5-18")

function Find-ReparsePoint {
    # Walks the tree without following links and returns the first reparse
    # point (junction, symlink, mount point), or $null.
    param([string]$Root)
    $Pending = New-Object System.Collections.Generic.Stack[string]
    $Pending.Push($Root)
    while ($Pending.Count -gt 0) {
        foreach ($Entry in Get-ChildItem -LiteralPath $Pending.Pop() -Force) {
            if ($Entry.Attributes -band [IO.FileAttributes]::ReparsePoint) {
                return $Entry.FullName
            }
            if ($Entry.PSIsContainer) {
                $Pending.Push($Entry.FullName)
            }
        }
    }
    return $null
}

function Assert-TrustedDataDirectory {
    # Refuses a directory SYSTEM and Administrators do not own, or one that
    # contains a link: a non-admin can create %ProgramData%\Seed ahead of the
    # installer, and the service writes its TLS key and JWT secret there.
    param([string]$Path)
    $Item = Get-Item -LiteralPath $Path -Force
    if ($Item.Attributes -band [IO.FileAttributes]::ReparsePoint) {
        throw "Refusing to use $Path as the Seed data directory: it is a reparse point. Remove it and run install.ps1 again."
    }
    $Owner = (Get-Acl -LiteralPath $Path).GetOwner([Security.Principal.SecurityIdentifier])
    if ($Owner.Value -ne $SystemSid.Value -and $Owner.Value -ne $AdministratorsSid.Value) {
        throw "Refusing to use $Path as the Seed data directory: it is owned by $Owner, not SYSTEM or Administrators. Delete it, or take ownership after checking its contents, then run install.ps1 again."
    }
    $Link = Find-ReparsePoint -Root $Path
    if ($null -ne $Link) {
        throw "Refusing to use $Path as the Seed data directory: $Link is a reparse point. Remove it and run install.ps1 again."
    }
}

function Assert-ProtectedDataAcl {
    # The directory must grant exactly SYSTEM and Administrators full control
    # and inherit nothing, or the service's secrets are readable by someone else.
    param([string]$Path)
    $Acl = Get-Acl -LiteralPath $Path
    $Rules = @($Acl.GetAccessRules($true, $true, [Security.Principal.SecurityIdentifier]))
    $Expected = @(@($SystemSid.Value, $AdministratorsSid.Value) | Sort-Object)
    $Actual = @(@($Rules | ForEach-Object { $_.IdentityReference.Value }) | Sort-Object)
    $Full = [Security.AccessControl.FileSystemRights]::FullControl
    $Allow = [Security.AccessControl.AccessControlType]::Allow
    $Problem = $null
    if (-not $Acl.AreAccessRulesProtected) {
        $Problem = "it inherits permissions from its parent"
    } elseif ($Actual.Count -ne $Expected.Count -or @(Compare-Object $Expected $Actual).Count -ne 0) {
        $Problem = "its access list is $($Actual -join ', '), not exactly SYSTEM and Administrators"
    } elseif (@($Rules | Where-Object { $_.FileSystemRights -ne $Full -or $_.AccessControlType -ne $Allow }).Count -ne 0) {
        $Problem = "an entry is not an Allow of Full Control"
    }
    if ($null -ne $Problem) {
        throw "Refusing to use $Path as the Seed data directory: $Problem. Delete it and run install.ps1 again."
    }
}

function Initialize-SeedDataDirectory {
    # Creates the data directory with a protected ACL, or accepts an existing
    # one only if it is already trustworthy. It either ends up exactly right
    # or the install stops before anything else has changed.
    param([string]$Path)
    $Created = $false
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path | Out-Null
        $Created = $true
    }
    # Also runs on a directory created a moment ago: another user can win the
    # race to create it, and then it is theirs, not ours.
    Assert-TrustedDataDirectory -Path $Path
    if ($Created) {
        $Security = New-Object Security.AccessControl.DirectorySecurity
        $Security.SetAccessRuleProtection($true, $false)
        $Security.SetOwner($AdministratorsSid)
        $Inherit = [Security.AccessControl.InheritanceFlags]"ContainerInherit, ObjectInherit"
        foreach ($Sid in @($SystemSid, $AdministratorsSid)) {
            $Security.AddAccessRule((New-Object Security.AccessControl.FileSystemAccessRule(
                $Sid, "FullControl", $Inherit, "None", "Allow")))
        }
        Set-Acl -LiteralPath $Path -AclObject $Security
    }
    Assert-ProtectedDataAcl -Path $Path
}

$Principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $Principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Error "Seed installation requires an elevated PowerShell session. Reopen PowerShell as Administrator and run install.ps1 again."
    exit 1
}

if (-not (Test-Path $SourceBinary)) {
    Write-Error "Cannot find $BinaryName next to install.ps1. Extract the complete archive before installing."
    exit 1
}

# Before anything is stopped or copied, so a refusal leaves the machine as it was.
Initialize-SeedDataDirectory -Path $DataDir

Write-Host "Installing Seed..." -ForegroundColor Green

$ExistingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($null -ne $ExistingService -and $ExistingService.Status -ne "Stopped") {
    Write-Host "  Stopping existing Windows service..."
    Stop-Service -Name $ServiceName -Force
    $ExistingService.WaitForStatus("Stopped", [TimeSpan]::FromSeconds(30))
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Copy-Item $SourceBinary $InstalledBinary -Force

$MachinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
# Whole entries, not a substring: "C:\Program Files\SeedTools" must not count
# as "C:\Program Files\Seed". -contains ignores case, as Windows paths do.
$PathEntries = @($MachinePath -split ";" | Where-Object { $_ } | ForEach-Object { $_.TrimEnd("\") })
if ($PathEntries -notcontains $InstallDir.TrimEnd("\")) {
    [Environment]::SetEnvironmentVariable("Path", "$($MachinePath.TrimEnd(';'));$InstallDir", "Machine")
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
# Domain and Private only: a laptop on a public network should not expose the
# diagnostics UI. A rule left by an earlier install is narrowed to match.
if ($null -eq (Get-NetFirewallRule -DisplayName "Seed" -ErrorAction SilentlyContinue)) {
    New-NetFirewallRule -DisplayName "Seed" `
        -Direction Inbound -Protocol TCP -LocalPort 8443 `
        -Profile Domain,Private `
        -Action Allow | Out-Null
} else {
    Set-NetFirewallRule -DisplayName "Seed" -Profile Domain,Private
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
Write-Host "  Data:    $DataDir"
Write-Host ""
Write-Host "Commands:"
Write-Host "  seed service status   - Check service status"
Write-Host "  seed service stop     - Stop service"
Write-Host "  seed service start    - Start service"
