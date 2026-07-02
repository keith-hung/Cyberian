#Requires -Version 7.0
#
# slip end-to-end test — Windows (PowerShell 7+, for .NET AF_UNIX support).
#
# The daemon and child are exercised for real. The `slip set` prompt reads the
# console (CONIN$) with echo off, which cannot be automated from a script, so
# this test plays the wire role of `slip set` with a .NET AF_UNIX client — the
# exact bytes `slip set` would send — and separately checks `slip set`'s
# no-daemon error path. A manual `slip set` check is printed at the end.
#
# Requires: PowerShell 7+ (Windows 10 1803+ / Server 2019+ for AF_UNIX). The slip
# binary is built automatically if `go` is available, or pass -Slip <path>.
#
# Usage:  pwsh -File .\test\e2e.ps1 [-Slip C:\path\to\slip.exe]

param([string]$Slip)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$SlipDir   = Split-Path -Parent $ScriptDir

# 1. Locate or build the binary.
if (-not $Slip) {
    $cand = Join-Path $SlipDir "slip.exe"
    if (Test-Path $cand) {
        $Slip = $cand
    } elseif (Get-Command go -ErrorAction SilentlyContinue) {
        Write-Host "building slip..."
        Push-Location $SlipDir; & go build -o slip.exe .; Pop-Location
        $Slip = $cand
    } else {
        Write-Error "no slip.exe and no 'go' to build one; pass -Slip <path>"; exit 1
    }
}

# Isolate sockets under a short temp LOCALAPPDATA so the runtime dir is controlled
# and the AF_UNIX socket path stays under the ~108-byte limit.
$Runtime = Join-Path $env:TEMP ("slip-e2e-" + [System.IO.Path]::GetRandomFileName().Substring(0, 6))
New-Item -ItemType Directory -Path $Runtime -Force | Out-Null
$env:LOCALAPPDATA = $Runtime
Remove-Item Env:\XDG_RUNTIME_DIR -ErrorAction SilentlyContinue  # force the Windows branch

$script:pass = 0; $script:fail = 0
function Ok($m) { Write-Host "  PASS $m"; $script:pass++ }
function No($m) { Write-Host "  FAIL $m"; $script:fail++ }

function Start-Slip([string[]]$SlipArgs) {
    $psi = [System.Diagnostics.ProcessStartInfo]::new()
    $psi.FileName = $Slip
    foreach ($a in $SlipArgs) { $psi.ArgumentList.Add($a) }
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true
    $psi.UseShellExecute = $false
    return [System.Diagnostics.Process]::Start($psi)
}

# Send bytes to an AF_UNIX socket the way `slip set` would: write, then close.
function Send-Unix([string]$Path, [byte[]]$Bytes) {
    $sock = [System.Net.Sockets.Socket]::new(
        [System.Net.Sockets.AddressFamily]::Unix,
        [System.Net.Sockets.SocketType]::Stream,
        [System.Net.Sockets.ProtocolType]::Unspecified)
    $sock.Connect([System.Net.Sockets.UnixDomainSocketEndPoint]::new($Path))
    [void]$sock.Send($Bytes)
    $sock.Shutdown([System.Net.Sockets.SocketShutdown]::Both)
    $sock.Close()
}

$SockDir = Join-Path $Runtime "slip"

Write-Host "== Test A: happy path (daemon -> AF_UNIX send -> child) =="
$secret = "s3cr3t-value"  # 12 chars; the child reports only the length
# Child reads all of stdin and prints its length, then exits 5.
$childCmd = '$x=[Console]::In.ReadToEnd(); Write-Output ("child-len=" + $x.Length); exit 5'
$p = Start-Slip @("daemon", "--timeout", "20", "--", "pwsh", "-NoProfile", "-Command", $childCmd)
$id = $p.StandardOutput.ReadLine()   # blocks until the daemon prints the ID
if ($id -notmatch '^\d{5}$') {
    No "daemon did not print a numeric ID (got '$id')"
} else {
    $sock = Join-Path $SockDir "$id.sock"
    Send-Unix $sock ([System.Text.Encoding]::UTF8.GetBytes($secret))
    $p.WaitForExit()
    $out = $p.StandardOutput.ReadToEnd(); $err = $p.StandardError.ReadToEnd()
    $problems = @()
    if ($p.ExitCode -ne 5) { $problems += "daemon exit=$($p.ExitCode), want 5" }
    if ($out -notmatch "child-len=$($secret.Length)") { $problems += "child did not receive the secret (stdout=$out)" }
    if (($out -match [regex]::Escape($secret)) -or ($err -match [regex]::Escape($secret))) { $problems += "secret LEAKED" }
    if (Test-Path $sock) { $problems += "socket not cleaned up" }
    if ($problems.Count) { $problems | ForEach-Object { Write-Host "  - $_" }; No "happy path" }
    else { Ok "happy path (exit=$($p.ExitCode), secret not leaked)" }
}

Write-Host "== Test B: timeout with no value =="
$p = Start-Slip @("daemon", "--timeout", "1", "--", "pwsh", "-NoProfile", "-Command", "Write-Output should-not-run")
$id = $p.StandardOutput.ReadLine()
$p.WaitForExit()
$out = $p.StandardOutput.ReadToEnd()
$sock = Join-Path $SockDir "$id.sock"
if (($p.ExitCode -ne 0) -and $id -and ($out -notmatch "should-not-run") -and -not (Test-Path $sock)) {
    Ok "timeout (exit=$($p.ExitCode), only ID on stdout, socket cleaned)"
} else {
    No "timeout (exit=$($p.ExitCode), id='$id', socket present=$([bool](Test-Path $sock)))"
}

Write-Host "== Test C: 'slip set' with no daemon =="
$err = (& $Slip set 00000) 2>&1 | Out-String
if (($LASTEXITCODE -ne 0) -and ($err -match "no daemon listening")) {
    Ok "no-daemon set errors clearly (exit=$LASTEXITCODE)"
} else {
    No "no-daemon set: exit=$LASTEXITCODE msg=$($err.Trim())"
}

Remove-Item -Recurse -Force $Runtime -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "Manual check (interactive tty / echo-off):"
Write-Host "  Console 1:  $Slip daemon --timeout 60 -- pwsh -NoProfile -Command `"Write-Output ('len=' + [Console]::In.ReadToEnd().Length)`""
Write-Host "  Console 2:  $Slip set <ID>   # type a secret; it must stay hidden, then 'len=' should match"
Write-Host ""
Write-Host "Result: $($script:pass) passed, $($script:fail) failed"
if ($script:fail -ne 0) { exit 1 }
