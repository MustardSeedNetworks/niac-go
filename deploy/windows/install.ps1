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

# Accounts that may own and write NiAC's data. Everything the service trusts
# (certificates, the attachment policy) lives under one directory that only
# these two can touch.
$SidSystem = New-Object Security.Principal.SecurityIdentifier("S-1-5-18")
$SidAdmins = New-Object Security.Principal.SecurityIdentifier("S-1-5-32-544")
$SidUsers = New-Object Security.Principal.SecurityIdentifier("S-1-5-32-545")
$TrustedSids = @($SidSystem.Value, $SidAdmins.Value)

# Every right that lets an account change a file or take it over.
$WriteRights = [Security.AccessControl.FileSystemRights]"WriteData, AppendData, WriteExtendedAttributes, WriteAttributes, Delete, ChangePermissions, TakeOwnership"

# Not %ProgramData%: an environment variable is whatever the session says, and
# this path decides where the service keeps its key.
$DataDir = Join-Path ([Environment]::GetFolderPath("CommonApplicationData")) "NiAC"
$PolicyFile = Join-Path $DataDir "attachment-policy.conf"

function Exit-WithError {
    param([string]$Message)
    Write-Error $Message
    exit 1
}

function Test-ReparsePoint {
    param([IO.FileSystemInfo]$Item)
    return (($Item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0)
}

function Get-OwnerSid {
    param([string]$Path)
    return (Get-Acl -LiteralPath $Path).GetOwner([Security.Principal.SecurityIdentifier]).Value
}

# The first account outside SYSTEM and Administrators that an allow entry lets
# write the item, or $null.
function Get-UntrustedWriter {
    param([string]$Path)
    $acl = Get-Acl -LiteralPath $Path
    foreach ($rule in $acl.GetAccessRules($true, $true, [Security.Principal.SecurityIdentifier])) {
        if ($rule.AccessControlType -ne [Security.AccessControl.AccessControlType]::Allow) { continue }
        if (($rule.FileSystemRights -band $WriteRights) -eq 0) { continue }
        if ($TrustedSids -contains $rule.IdentityReference.Value) { continue }
        return $rule.IdentityReference.Value
    }
    return $null
}

# Every entry below $Root, without following links: a reparse point is returned
# but never entered.
function Get-DataDirEntry {
    param([string]$Root)
    $pending = New-Object Collections.Generic.Stack[string]
    $pending.Push($Root)
    while ($pending.Count -gt 0) {
        $directory = $pending.Pop()
        foreach ($entry in Get-ChildItem -LiteralPath $directory -Force) {
            $entry
            if ($entry.PSIsContainer -and -not (Test-ReparsePoint -Item $entry)) {
                $pending.Push($entry.FullName)
            }
        }
    }
}

# SYSTEM and Administrators full control, nothing inherited, Administrators own.
function Get-DataDirSecurity {
    $security = New-Object Security.AccessControl.DirectorySecurity
    $security.SetAccessRuleProtection($true, $false)
    $security.SetOwner($SidAdmins)
    $inherit = [Security.AccessControl.InheritanceFlags]"ContainerInherit, ObjectInherit"
    foreach ($sid in @($SidSystem, $SidAdmins)) {
        $security.AddAccessRule((New-Object Security.AccessControl.FileSystemAccessRule(
            $sid, [Security.AccessControl.FileSystemRights]::FullControl, $inherit,
            [Security.AccessControl.PropagationFlags]::None, [Security.AccessControl.AccessControlType]::Allow)))
    }
    return $security
}

# Same, for the policy file: everyone may read it, only the operators write.
function Get-PolicyFileSecurity {
    $security = New-Object Security.AccessControl.FileSecurity
    $security.SetAccessRuleProtection($true, $false)
    $security.SetOwner($SidAdmins)
    $grants = @(
        @($SidSystem, [Security.AccessControl.FileSystemRights]::FullControl),
        @($SidAdmins, [Security.AccessControl.FileSystemRights]::FullControl),
        @($SidUsers, [Security.AccessControl.FileSystemRights]::Read)
    )
    foreach ($grant in $grants) {
        $security.AddAccessRule((New-Object Security.AccessControl.FileSystemAccessRule(
            $grant[0], $grant[1], [Security.AccessControl.AccessControlType]::Allow)))
    }
    return $security
}

# Creates or takes over %ProgramData%\NiAC. It fails closed: a directory that
# belongs to someone else, or holds a link, is never adopted, because the
# service keeps its private key there.
function Initialize-DataDir {
    $item = Get-Item -LiteralPath $DataDir -Force -ErrorAction SilentlyContinue
    if ($null -ne $item) {
        if (-not $item.PSIsContainer) {
            Exit-WithError "Refusing to install: $DataDir is a file. Remove it and run install.ps1 again."
        }
        if (Test-ReparsePoint -Item $item) {
            Exit-WithError "Refusing to install: $DataDir is a junction or symbolic link. Remove it and run install.ps1 again."
        }
        $owner = Get-OwnerSid -Path $DataDir
        if ($TrustedSids -notcontains $owner) {
            Exit-WithError "Refusing to install: $DataDir is owned by $owner, not SYSTEM or Administrators. Another account created it; review it, remove it and run install.ps1 again."
        }
        # The operator's policy is obeyed at startup, so a copy someone else
        # could write must be reviewed, not adopted along with the directory.
        if (Test-Path -LiteralPath $PolicyFile) {
            $writer = Get-UntrustedWriter -Path $PolicyFile
            if ($null -ne $writer) {
                Exit-WithError "Refusing to install: $PolicyFile is writable by $writer. Review its contents, remove it and run install.ps1 again."
            }
        }
    } else {
        New-Item -ItemType Directory -Path $DataDir | Out-Null
    }

    # One Set-Acl with the complete ACL, before anything is read or trusted.
    Set-Acl -LiteralPath $DataDir -AclObject (Get-DataDirSecurity)

    # Locked, so nothing can change under the walk. Whatever it finds was there
    # before: refuse links and entries someone else owns, and let the rest
    # inherit the directory's ACL.
    foreach ($entry in @(Get-DataDirEntry -Root $DataDir)) {
        if (Test-ReparsePoint -Item $entry) {
            Exit-WithError "Refusing to install: $($entry.FullName) is a junction or symbolic link. Remove it and run install.ps1 again."
        }
        $entryOwner = Get-OwnerSid -Path $entry.FullName
        if ($TrustedSids -notcontains $entryOwner) {
            Exit-WithError "Refusing to install: $($entry.FullName) is owned by $entryOwner, not SYSTEM or Administrators. Review it, remove it and run install.ps1 again."
        }
        & icacls.exe $entry.FullName /reset /C /Q | Out-Null
        if ($LASTEXITCODE -ne 0) {
            Exit-WithError "Could not reset the ACL on $($entry.FullName) (icacls exit code $LASTEXITCODE)."
        }
    }

    Assert-DataDir
}

function Assert-DataDir {
    $acl = Get-Acl -LiteralPath $DataDir
    if (-not $acl.AreAccessRulesProtected) {
        Exit-WithError "$DataDir still inherits its ACL."
    }
    if ($acl.GetOwner([Security.Principal.SecurityIdentifier]).Value -ne $SidAdmins.Value) {
        Exit-WithError "$DataDir is not owned by Administrators."
    }
    $rules = @($acl.GetAccessRules($true, $true, [Security.Principal.SecurityIdentifier]))
    $granted = @($rules | ForEach-Object { $_.IdentityReference.Value } | Sort-Object)
    if (($rules.Count -ne 2) -or (($granted -join ",") -ne (($TrustedSids | Sort-Object) -join ","))) {
        Exit-WithError "$DataDir must grant SYSTEM and Administrators only."
    }
    foreach ($rule in $rules) {
        if (($rule.AccessControlType -ne [Security.AccessControl.AccessControlType]::Allow) -or
            ($rule.FileSystemRights -ne [Security.AccessControl.FileSystemRights]::FullControl) -or
            $rule.IsInherited) {
            Exit-WithError "$DataDir has an unexpected entry for $($rule.IdentityReference.Value)."
        }
    }
    foreach ($entry in @(Get-DataDirEntry -Root $DataDir)) {
        if ((Get-Acl -LiteralPath $entry.FullName).AreAccessRulesProtected) {
            Exit-WithError "$($entry.FullName) does not inherit the directory's ACL."
        }
    }
}

# The operator's attachment policy file: created once and never overwritten, so
# a reinstall keeps what the operator approved. Only SYSTEM and Administrators
# may write it (everyone may read it); the daemon ignores it otherwise.
function Initialize-PolicyFile {
    if (-not (Test-Path -LiteralPath $PolicyFile)) {
        # Built under the directory's protected ACL, given its own protected ACL
        # while still empty, then renamed into place: the rename keeps the ACL,
        # so the policy never exists under anything wider.
        $staged = Join-Path $DataDir ("attachment-policy.conf." + [Guid]::NewGuid().ToString("N") + ".new")
        try {
            New-Item -ItemType File -Path $staged | Out-Null
            Set-Acl -LiteralPath $staged -AclObject (Get-PolicyFileSecurity)
            Set-Content -LiteralPath $staged -Encoding ascii -Value @(
                "# Physical attachments the NIAC daemon approves, one INTERFACE=MODE per line",
                "# (direct, access:VLAN or trunk:VLAN,...). Manage it with: niac attachment-policy"
            )
            [IO.File]::Move($staged, $PolicyFile)
        } finally {
            if (Test-Path -LiteralPath $staged) { Remove-Item -LiteralPath $staged -Force }
        }
    } else {
        # Initialize-DataDir reset it to inherit; give it back its own ACL.
        Set-Acl -LiteralPath $PolicyFile -AclObject (Get-PolicyFileSecurity)
    }

    $item = Get-Item -LiteralPath $PolicyFile -Force
    if ($item.PSIsContainer -or (Test-ReparsePoint -Item $item)) {
        Exit-WithError "$PolicyFile is not a regular file."
    }
    $acl = Get-Acl -LiteralPath $PolicyFile
    if (-not $acl.AreAccessRulesProtected) {
        Exit-WithError "$PolicyFile still inherits its ACL."
    }
    if ($TrustedSids -notcontains $acl.GetOwner([Security.Principal.SecurityIdentifier]).Value) {
        Exit-WithError "$PolicyFile is not owned by SYSTEM or Administrators."
    }
    $writer = Get-UntrustedWriter -Path $PolicyFile
    if ($null -ne $writer) {
        Exit-WithError "$PolicyFile is writable by $writer."
    }
}

# True when the listener address binds loopback only. An empty host (":8445")
# binds every interface; an empty address is the loopback default.
function Test-LoopbackListen {
    param([string]$Address)
    if ([string]::IsNullOrWhiteSpace($Address)) { return $true }
    $listenHost = $Address
    if ($Address -match '^\[(.+)\]:\d+$') { $listenHost = $Matches[1] }
    elseif ($Address -match '^(.*):\d+$') { $listenHost = $Matches[1] }
    if ($listenHost -eq "localhost") { return $true }
    $parsed = $null
    if ([Net.IPAddress]::TryParse($listenHost, [ref]$parsed)) { return [Net.IPAddress]::IsLoopback($parsed) }
    return $false
}

function Get-ListenPort {
    param([string]$Address)
    if ($Address -match ':(\d+)$') { return [int]$Matches[1] }
    return 8445
}

Write-Host "Installing NiAC..." -ForegroundColor Green

Initialize-DataDir

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

Initialize-PolicyFile

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

# The service defaults to loopback, which needs no rule. A rule is made only
# when NIAC_LISTEN_ADDR (read at Machine scope, where the service reads it)
# binds beyond loopback, and then only for networks the operator manages.
$ListenAddress = [Environment]::GetEnvironmentVariable("NIAC_LISTEN_ADDR", "Machine")
Get-NetFirewallRule -DisplayName "NiAC Simulator" -ErrorAction SilentlyContinue | Remove-NetFirewallRule
if (Test-LoopbackListen -Address $ListenAddress) {
    Write-Host "  Listener is loopback only; no firewall rule needed."
} else {
    $ListenPort = Get-ListenPort -Address $ListenAddress
    Write-Host "  Adding firewall rule for port $ListenPort (Domain and Private networks)..."
    New-NetFirewallRule -DisplayName "NiAC Simulator" `
        -Direction Inbound -Protocol TCP -LocalPort $ListenPort `
        -Profile Domain, Private -Action Allow | Out-Null
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
