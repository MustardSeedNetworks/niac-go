# NiAC - Windows Installation Script
# Run as Administrator from an extracted NIAC Windows archive.

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$InstallDir = "$env:ProgramFiles\NiAC"
$BinaryName = "niac.exe"
$ServiceName = "NiACSimulator"
$SourceBinary = Join-Path $ScriptDir $BinaryName
$InstalledBinary = Join-Path $InstallDir $BinaryName

$Principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $Principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Error "NiAC installation requires an elevated PowerShell session. Reopen PowerShell as Administrator and run install.ps1 again."
    exit 1
}

if (-not (Test-Path $SourceBinary)) {
    Write-Error "Cannot find $BinaryName next to install.ps1. Extract the complete archive before installing."
    exit 1
}

Write-Host "Installing NiAC..." -ForegroundColor Green

$ExistingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($null -ne $ExistingService -and $ExistingService.Status -ne "Stopped") {
    Write-Host "  Stopping existing Windows service..."
    Stop-Service -Name $ServiceName -Force
    $ExistingService.WaitForStatus("Stopped", [TimeSpan]::FromSeconds(30))
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
New-Item -ItemType Directory -Force -Path "$InstallDir\data" | Out-Null
New-Item -ItemType Directory -Force -Path "$InstallDir\logs" | Out-Null
Copy-Item $SourceBinary $InstalledBinary -Force

$MachinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
if ($MachinePath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$MachinePath;$InstallDir", "Machine")
    Write-Host "  Added $InstallDir to system PATH"
}

# The operator's attachment policy file: created once and never overwritten, so
# a reinstall keeps what the operator approved. Only SYSTEM and Administrators
# may write it (everyone may read it); the daemon ignores it otherwise.
$PolicyDir = Join-Path $env:ProgramData "NiAC"
$PolicyFile = Join-Path $PolicyDir "attachment-policy.conf"
if (-not (Test-Path $PolicyFile)) {
    New-Item -ItemType Directory -Force -Path $PolicyDir | Out-Null
    Set-Content -Path $PolicyFile -Encoding ascii -Value @(
        "# Physical attachments the NIAC daemon approves, one INTERFACE=MODE per line",
        "# (direct, access:VLAN or trunk:VLAN,...). Manage it with: niac attachment-policy"
    )
    & icacls.exe $PolicyFile /inheritance:r /grant:r "*S-1-5-18:(F)" "*S-1-5-32-544:(F)" "*S-1-5-32-545:(R)" | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Remove-Item -Force $PolicyFile
        Write-Error "Could not restrict $PolicyFile to SYSTEM and Administrators (icacls exit code $LASTEXITCODE)."
        exit $LASTEXITCODE
    }
}

Write-Host "  Installing Windows service..."
if ($null -eq $ExistingService) {
    & $InstalledBinary service install
    if ($LASTEXITCODE -ne 0) {
        $ExitCode = $LASTEXITCODE
        Write-Error "NiAC service installation failed with exit code $ExitCode."
        exit $ExitCode
    }
} else {
    # Not sc.exe: Windows PowerShell 5.1 drops the embedded quotes around a
    # path with spaces when it builds a native command line (niac-go#2699).
    $Service = Get-CimInstance Win32_Service -Filter "Name='$ServiceName'"
    $Result = Invoke-CimMethod -InputObject $Service -MethodName Change `
        -Arguments @{ PathName = "`"$InstalledBinary`" service run" }
    if ($Result.ReturnValue -ne 0) {
        $ExitCode = $Result.ReturnValue
        Write-Error "NiAC service reconfiguration failed with Win32_Service.Change result $ExitCode."
        exit $ExitCode
    }
}

Write-Host "  Adding firewall rule for port 8445..."
if ($null -eq (Get-NetFirewallRule -DisplayName "NiAC Simulator" -ErrorAction SilentlyContinue)) {
    New-NetFirewallRule -DisplayName "NiAC Simulator" `
        -Direction Inbound -Protocol TCP -LocalPort 8445 `
        -Action Allow | Out-Null
}

Write-Host "  Starting service..."
& $InstalledBinary service start
if ($LASTEXITCODE -ne 0) {
    $ExitCode = $LASTEXITCODE
    Write-Error "NiAC service startup failed with exit code $ExitCode."
    exit $ExitCode
}

Write-Host ""
Write-Host "Installation complete!" -ForegroundColor Green
Write-Host "  Web UI: https://localhost:8445"
Write-Host "  Service: $ServiceName"
Write-Host ""
Write-Host "Commands:"
Write-Host "  niac service status   - Check service status"
Write-Host "  niac service stop     - Stop service"
Write-Host "  niac service start    - Start service"
