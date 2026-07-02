#Requires -Version 5.1
$ErrorActionPreference = "Stop"

# slip is cross-platform (AF_UNIX on Linux/macOS/Windows). This launcher serves
# native Windows; slip-launcher.sh serves Linux/macOS/WSL. The slip binary itself
# needs Windows 10 1803+ / Server 2019+ (AF_UNIX support) at runtime.

$Repo = "keith-hung/Cyberian"
$Version = "v0.3.2"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$CacheDir = Join-Path $ScriptDir ".cache"
$Bin = Join-Path $CacheDir "slip.exe"

# Fast path: binary already cached
if (Test-Path $Bin) {
    & $Bin @args
    exit $LASTEXITCODE
}

# Detect architecture
$Arch = $null
try {
    $Arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
        "X64"   { "amd64" }
        "Arm64" { "arm64" }
    }
} catch {}

if (-not $Arch) {
    $Arch = switch ($env:PROCESSOR_ARCHITECTURE) {
        "AMD64" { "amd64" }
        "ARM64" { "arm64" }
        default {
            [Console]::Error.WriteLine("slip: unsupported architecture: $_")
            exit 1
        }
    }
}

# Download binary
$Url = "https://github.com/$Repo/releases/download/$Version/slip_windows_${Arch}.exe"
if (-not (Test-Path $CacheDir)) {
    New-Item -ItemType Directory -Path $CacheDir -Force | Out-Null
}

[Console]::Error.WriteLine("Downloading slip $Version for windows/${Arch}...")
try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $Url -OutFile $Bin -UseBasicParsing
} catch {
    [Console]::Error.WriteLine("slip: download failed: " + $_.Exception.Message)
    exit 1
}

[Console]::Error.WriteLine("Downloaded successfully.")
& $Bin @args
exit $LASTEXITCODE
