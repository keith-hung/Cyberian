#Requires -Version 5.1
$ErrorActionPreference = "Stop"

$Repo = "keith-hung/Cyberian"
$Version = "v0.1.0"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$CacheDir = Join-Path $ScriptDir ".cache"
$Bin = Join-Path $CacheDir "azuredevops-cli.exe"

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
            [Console]::Error.WriteLine('{"success":false,"error":"Unsupported architecture: ' + $_ + '"}')
            exit 1
        }
    }
}

# Download binary
$Url = "https://github.com/$Repo/releases/download/$Version/azuredevops-cli_windows_${Arch}.exe"
if (-not (Test-Path $CacheDir)) {
    New-Item -ItemType Directory -Path $CacheDir -Force | Out-Null
}

[Console]::Error.WriteLine("Downloading azuredevops-cli $Version for windows/${Arch}...")
try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $Url -OutFile $Bin -UseBasicParsing
} catch {
    [Console]::Error.WriteLine('{"success":false,"error":"Download failed: ' + $_.Exception.Message + '"}')
    exit 1
}

[Console]::Error.WriteLine("Downloaded successfully.")
& $Bin @args
exit $LASTEXITCODE
