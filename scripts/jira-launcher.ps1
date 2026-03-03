#Requires -Version 5.1
$ErrorActionPreference = "Stop"

$Repo = "ankitpokhrel/jira-cli"
$Version = "v1.7.0"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$CacheDir = Join-Path $ScriptDir ".cache"
$Bin = Join-Path $CacheDir "jira.exe"

# Fast path: binary already cached — still check auto-init
if (Test-Path $Bin) {
    $JiraConfig = Join-Path $env:USERPROFILE ".config\.jira\.config.yml"
    if ((-not (Test-Path $JiraConfig)) -and $env:JIRA_API_TOKEN -and $env:JIRA_SERVER_URL -and $env:JIRA_USER_EMAIL) {
        $InitArgs = @("init", "--installation", "cloud", "--server", $env:JIRA_SERVER_URL, "--login", $env:JIRA_USER_EMAIL, "--force")
        if ($env:JIRA_PROJECT) { $InitArgs += @("--project", $env:JIRA_PROJECT) }
        if ($env:JIRA_BOARD)   { $InitArgs += @("--board", $env:JIRA_BOARD) }

        [Console]::Error.WriteLine("Auto-initializing jira-cli config...")
        & $Bin @InitArgs 2>&1 | ForEach-Object { [Console]::Error.WriteLine($_) }
    }
    & $Bin @args
    exit $LASTEXITCODE
}

# Detect architecture — jira-cli only provides Windows x86_64
$Arch = $null
try {
    $Arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
        "X64" { "x86_64" }
    }
} catch {}

if (-not $Arch) {
    $Arch = switch ($env:PROCESSOR_ARCHITECTURE) {
        "AMD64" { "x86_64" }
    }
}

if (-not $Arch) {
    $DetectedArch = try { [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture } catch { $env:PROCESSOR_ARCHITECTURE }
    [Console]::Error.WriteLine('{"success":false,"error":"Unsupported architecture for jira-cli on Windows: ' + $DetectedArch + ' (only x86_64 is available)"}')
    exit 1
}

# Download and extract binary
$VersionNum = $Version.TrimStart("v")
$Archive = "jira_${VersionNum}_windows_${Arch}.zip"
$Url = "https://github.com/$Repo/releases/download/$Version/$Archive"

if (-not (Test-Path $CacheDir)) {
    New-Item -ItemType Directory -Path $CacheDir -Force | Out-Null
}

$TmpFile = Join-Path $CacheDir $Archive
$TmpExtractDir = Join-Path $CacheDir "_jira_extract"

[Console]::Error.WriteLine("Downloading jira-cli $Version for windows/${Arch}...")
try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $Url -OutFile $TmpFile -UseBasicParsing
} catch {
    [Console]::Error.WriteLine('{"success":false,"error":"Download failed: ' + $_.Exception.Message + '"}')
    exit 1
}

# Extract binary from zip
try {
    if (Test-Path $TmpExtractDir) {
        Remove-Item -Recurse -Force $TmpExtractDir
    }
    Expand-Archive -Path $TmpFile -DestinationPath $TmpExtractDir -Force

    $Found = Get-ChildItem -Path $TmpExtractDir -Recurse -Filter "jira.exe" | Select-Object -First 1
    if ($null -eq $Found) {
        throw "jira.exe not found in archive"
    }
    Copy-Item -Path $Found.FullName -Destination $Bin -Force
} catch {
    [Console]::Error.WriteLine('{"success":false,"error":"Failed to extract jira binary: ' + $_.Exception.Message + '"}')
    Remove-Item -Force -ErrorAction SilentlyContinue $TmpFile
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $TmpExtractDir
    exit 1
} finally {
    Remove-Item -Force -ErrorAction SilentlyContinue $TmpFile
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $TmpExtractDir
}

[Console]::Error.WriteLine("Downloaded successfully.")

# Auto-init if env vars are set and no config exists
$JiraConfig = Join-Path $env:USERPROFILE ".config\.jira\.config.yml"
if ((-not (Test-Path $JiraConfig)) -and $env:JIRA_API_TOKEN -and $env:JIRA_SERVER_URL -and $env:JIRA_USER_EMAIL) {
    $InitArgs = @("init", "--installation", "cloud", "--server", $env:JIRA_SERVER_URL, "--login", $env:JIRA_USER_EMAIL, "--force")
    if ($env:JIRA_PROJECT) { $InitArgs += @("--project", $env:JIRA_PROJECT) }
    if ($env:JIRA_BOARD)   { $InitArgs += @("--board", $env:JIRA_BOARD) }

    [Console]::Error.WriteLine("Auto-initializing jira-cli config...")
    & $Bin @InitArgs 2>&1 | ForEach-Object { [Console]::Error.WriteLine($_) }
}

& $Bin @args
exit $LASTEXITCODE
