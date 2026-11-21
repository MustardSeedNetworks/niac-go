# CI harness for install.ps1. Run it elevated on a disposable Windows runner:
# it creates and deletes %ProgramData%\NiAC, installs the NiACSimulator
# service and removes it again.
#
#   .\ci-install-test.ps1 -Archive <extracted release archive> -ExpectedVersion <version>

param(
    [Parameter(Mandatory = $true)][string]$Archive,
    [Parameter(Mandatory = $true)][string]$ExpectedVersion
)

$ErrorActionPreference = "Stop"

$Installer = Join-Path $Archive "install.ps1"
$ServiceName = "NiACSimulator"
$DataDir = Join-Path ([Environment]::GetFolderPath("CommonApplicationData")) "NiAC"
$InstalledBinary = Join-Path ([Environment]::GetFolderPath("ProgramFiles")) "NiAC\niac.exe"
$Scratch = Join-Path ([IO.Path]::GetTempPath()) "niac-install-test"
$Sid = @{ System = "S-1-5-18"; Admins = "S-1-5-32-544"; Users = "S-1-5-32-545" }

function Invoke-Installer {
    # Windows PowerShell 5.1 is what operators run; native stderr must not turn
    # into a terminating error here, or a refusal could not be inspected.
    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $output = & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $Installer 2>&1 | Out-String
    $code = $LASTEXITCODE
    $ErrorActionPreference = $previous
    return [pscustomobject]@{ ExitCode = $code; Output = $output }
}

function Invoke-Native {
    param([string]$File, [string[]]$Arguments)
    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $output = & $File @Arguments 2>&1 | Out-String
    $code = $LASTEXITCODE
    $ErrorActionPreference = $previous
    if ($code -ne 0) { throw "$File $($Arguments -join ' ') exited $code`: $output" }
}

function Reset-DataDir {
    $item = Get-Item -LiteralPath $DataDir -Force -ErrorAction SilentlyContinue
    if ($null -eq $item) { return }
    if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) {
        Invoke-Native -File "cmd.exe" -Arguments @("/c", "rmdir", $DataDir)
        return
    }
    foreach ($child in Get-ChildItem -LiteralPath $DataDir -Force) {
        if ($child.Attributes -band [IO.FileAttributes]::ReparsePoint) {
            Invoke-Native -File "cmd.exe" -Arguments @("/c", "rmdir", $child.FullName)
        }
    }
    Invoke-Native -File "takeown.exe" -Arguments @("/F", $DataDir, "/R", "/D", "Y")
    Invoke-Native -File "icacls.exe" -Arguments @($DataDir, "/grant", "*$($Sid.Admins):(OI)(CI)F", "/T", "/C", "/Q")
    Remove-Item -LiteralPath $DataDir -Recurse -Force
}

function Assert-InstallRefuses {
    param([string]$Case, [string[]]$Expect)
    $result = Invoke-Installer
    if ($result.ExitCode -eq 0) { throw "${Case}: install.ps1 succeeded; it must refuse" }
    # The child wraps its error text at the console width, so compare with
    # whitespace removed from both sides.
    $flat = $result.Output -replace '\s', ''
    foreach ($fragment in $Expect) {
        if (-not $flat.Contains(($fragment -replace '\s', ''))) {
            throw "${Case}: the refusal does not mention '$fragment':`n$($result.Output)"
        }
    }
    if ($null -ne (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue)) {
        throw "${Case}: the refusal left a service behind"
    }
    Write-Host "ok: $Case is refused"
    Reset-DataDir
}

function Set-Owner {
    param([string]$Path, [string]$OwnerSid)
    Invoke-Native -File "icacls.exe" -Arguments @($Path, "/setowner", "*$OwnerSid")
}

function Get-SidList {
    param($Acl)
    return @($Acl.GetAccessRules($true, $true, [Security.Principal.SecurityIdentifier]) |
        ForEach-Object { $_.IdentityReference.Value } | Sort-Object -Unique)
}

function Assert-Acl {
    param([string]$Path, [string[]]$ExpectedSids, [bool]$Protected, [string[]]$AllowedOwners)
    $acl = Get-Acl -LiteralPath $Path
    $sids = (Get-SidList -Acl $acl) -join ","
    $want = ($ExpectedSids | Sort-Object) -join ","
    if ($sids -ne $want) { throw "${Path}: ACL grants $sids, want exactly $want" }
    if ($acl.AreAccessRulesProtected -ne $Protected) {
        throw "${Path}: AreAccessRulesProtected is $($acl.AreAccessRulesProtected), want $Protected"
    }
    $owner = $acl.GetOwner([Security.Principal.SecurityIdentifier]).Value
    if ($AllowedOwners -notcontains $owner) { throw "${Path}: owned by $owner" }
    Write-Host "ok: $Path grants $sids, protected=$Protected, owner=$owner"
}

function Assert-Installed {
    $service = Get-Service -Name $ServiceName
    if ($service.Status -ne "Running") { throw "service is $($service.Status), want Running" }

    $answer = $null
    for ($attempt = 0; $attempt -lt 60 -and $null -eq $answer; $attempt++) {
        $body = & curl.exe --silent --insecure --max-time 5 https://127.0.0.1:8445/__version
        if ($LASTEXITCODE -eq 0 -and $body) { $answer = $body | ConvertFrom-Json } else { Start-Sleep -Seconds 1 }
    }
    if ($null -eq $answer) { throw "https://127.0.0.1:8445/__version did not answer within 60 seconds" }
    if ($answer.version -ne $ExpectedVersion) { throw "/__version reports $($answer.version), want $ExpectedVersion" }
    if (-not $answer.uiBuildHash) { throw "/__version reports an empty uiBuildHash" }
    Write-Host "ok: /__version answers $($answer.version) (uiBuildHash $($answer.uiBuildHash))"

    $operators = @($Sid.System, $Sid.Admins)
    Assert-Acl -Path $DataDir -ExpectedSids $operators -Protected $true -AllowedOwners @($Sid.Admins)
    Assert-Acl -Path (Join-Path $DataDir "attachment-policy.conf") `
        -ExpectedSids @($Sid.System, $Sid.Admins, $Sid.Users) -Protected $true -AllowedOwners @($Sid.Admins)
    # The service creates certs/ itself; it must have inherited the data dir's ACL.
    Assert-Acl -Path (Join-Path $DataDir "certs") -ExpectedSids $operators -Protected $false `
        -AllowedOwners @($Sid.System, $Sid.Admins)

    if ($null -ne (Get-NetFirewallRule -DisplayName "NiAC Simulator" -ErrorAction SilentlyContinue)) {
        throw "a loopback-only install must not open the firewall"
    }
    Write-Host "ok: no firewall rule for a loopback listener"
}

New-Item -ItemType Directory -Force -Path $Scratch | Out-Null
try {
    Reset-DataDir

    Write-Host "::group::refusals"
    $junctionTarget = Join-Path $Scratch "elsewhere"
    New-Item -ItemType Directory -Force -Path $junctionTarget | Out-Null

    New-Item -ItemType Directory -Path $DataDir | Out-Null
    Set-Owner -Path $DataDir -OwnerSid $Sid.Users
    Assert-InstallRefuses -Case "a directory owned by Users" -Expect @($DataDir, "owned by")

    Invoke-Native -File "cmd.exe" -Arguments @("/c", "mklink", "/J", $DataDir, $junctionTarget)
    Assert-InstallRefuses -Case "a junction in place of the directory" -Expect @($DataDir, "junction")

    New-Item -ItemType Directory -Path $DataDir | Out-Null
    Set-Owner -Path $DataDir -OwnerSid $Sid.Admins
    Invoke-Native -File "cmd.exe" -Arguments @("/c", "mklink", "/J", (Join-Path $DataDir "certs"), $junctionTarget)
    Assert-InstallRefuses -Case "a junction named certs" -Expect @("certs", "junction")

    New-Item -ItemType Directory -Path $DataDir | Out-Null
    Set-Owner -Path $DataDir -OwnerSid $Sid.Admins
    $planted = Join-Path $DataDir "planted.pem"
    New-Item -ItemType File -Path $planted | Out-Null
    Set-Owner -Path $planted -OwnerSid $Sid.Users
    Assert-InstallRefuses -Case "a file owned by Users" -Expect @("planted.pem", "owned by")

    New-Item -ItemType Directory -Path $DataDir | Out-Null
    Set-Owner -Path $DataDir -OwnerSid $Sid.Admins
    $policy = Join-Path $DataDir "attachment-policy.conf"
    New-Item -ItemType File -Path $policy | Out-Null
    Set-Owner -Path $policy -OwnerSid $Sid.Admins
    Invoke-Native -File "icacls.exe" -Arguments @($policy, "/grant", "*$($Sid.Users):(W)")
    Assert-InstallRefuses -Case "a policy file Users can write" -Expect @("attachment-policy.conf", "writable by")
    Write-Host "::endgroup::"

    Write-Host "::group::install"
    $first = Invoke-Installer
    Write-Host $first.Output
    if ($first.ExitCode -ne 0) { throw "install.ps1 exited $($first.ExitCode)" }
    Assert-Installed
    Write-Host "::endgroup::"

    Write-Host "::group::reinstall over an existing install"
    $second = Invoke-Installer
    Write-Host $second.Output
    if ($second.ExitCode -ne 0) { throw "a second install.ps1 exited $($second.ExitCode)" }
    Assert-Installed
    Write-Host "::endgroup::"
} finally {
    Write-Host "::group::uninstall"
    $ErrorActionPreference = "Continue"
    if (Test-Path -LiteralPath $InstalledBinary) {
        & $InstalledBinary service stop
        & $InstalledBinary service uninstall
    }
    Write-Host "::endgroup::"
    $ErrorActionPreference = "Stop"
    if ($null -ne (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue)) {
        throw "the $ServiceName service is still registered after uninstall"
    }
    Reset-DataDir
}
