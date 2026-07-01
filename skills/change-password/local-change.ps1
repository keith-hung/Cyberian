#Requires -Version 5.1
# Change the current user's on-prem AD password from a domain-joined Windows
# machine that can reach a domain controller (intranet or VPN). Same operation
# as Ctrl+Alt+Del "Change a password"; no OTP.
#
# Passwords are read from stdin (two lines: old, then new) and never appear on
# the command line. stdin-only: no interactive prompt fallback.
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$Domain,
    [Parameter(Mandatory = $true)][string]$User
)

$ErrorActionPreference = 'Stop'

function Write-Result($obj) { $obj | ConvertTo-Json -Compress }

if ([Console]::IsInputRedirected -eq $false) {
    Write-Result @{ success = $false; error = 'stdin required (pipe old and new password, one per line)' }
    exit 3
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
