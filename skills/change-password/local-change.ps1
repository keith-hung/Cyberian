#Requires -Version 5.1
# Change the current user's on-prem AD password from a domain-joined Windows
# machine that can reach a domain controller (intranet or VPN). Same operation
# as Ctrl+Alt+Del "Change a password"; no OTP.
#
# Two modes:
#   -Detect : print environment JSON (domainJoined / userIsDomain / dcReachable /
#             domain / user / adViable) and exit. Performs no change, reads no stdin.
#   (change): read old then new password from stdin (two lines) and change the
#             password via ADSI. stdin-only; no interactive prompt fallback.
#
# Domain/User default to the logged-in session ($env:USERDOMAIN / $env:USERNAME)
# and may be overridden with -Domain / -User. No environment-variable config.
[CmdletBinding()]
param(
    [switch]$Detect,
    [string]$Domain,
    [string]$User
)

$ErrorActionPreference = 'Stop'

function Write-Result($obj) { $obj | ConvertTo-Json -Compress }

# Resolve target account: explicit args override the logged-in session.
if (-not $Domain) { $Domain = $env:USERDOMAIN }
if (-not $User)   { $User   = $env:USERNAME }

# Environment detection, shared by -Detect and the change-mode guard.
$cs = Get-CimInstance Win32_ComputerSystem
$domainJoined = [bool]$cs.PartOfDomain
$userIsDomain = ($Domain -ne $env:COMPUTERNAME)
$dcReachable = $false
if ($domainJoined -and $userIsDomain) {
    & nltest /dsgetdc:$Domain *> $null
    $dcReachable = ($LASTEXITCODE -eq 0)
}
$adViable = ($domainJoined -and $userIsDomain -and $dcReachable)

if ($Detect) {
    Write-Result @{
        domainJoined = $domainJoined
        userIsDomain = $userIsDomain
        dcReachable  = $dcReachable
        domain       = $Domain
        user         = $User
        adViable     = $adViable
    }
    exit 0
}

# --- Change mode ---
if ([Console]::IsInputRedirected -eq $false) {
    Write-Result @{ success = $false; error = 'stdin required (pipe old and new password, one per line)' }
    exit 3
}
if (-not $domainJoined -or -not $userIsDomain) {
    Write-Result @{ success = $false; error = 'not a domain-joined session; use the chpw portal path instead' }
    exit 2
}

$old = [Console]::In.ReadLine()
$new = [Console]::In.ReadLine()
if ([string]::IsNullOrEmpty($old) -or [string]::IsNullOrEmpty($new)) {
    Write-Result @{ success = $false; error = 'expected two stdin lines: old password then new password' }
    exit 3
}

try {
    $entry = [ADSI]("WinNT://$Domain/$User,user")
    $entry.ChangePassword($old, $new)
    Write-Result @{ success = $true }
    exit 0
} catch {
    $msg = $_.Exception.Message
    if ($_.Exception.InnerException) { $msg = $_.Exception.InnerException.Message }
    Write-Result @{ success = $false; error = $msg }
    exit 1
}
