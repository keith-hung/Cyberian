#!/usr/bin/env bash
set -euo pipefail

# slip is cross-platform (AF_UNIX on Linux/macOS/Windows; Windows 10 1803+).
# This launcher serves Linux/macOS/WSL; slip-launcher.ps1 serves native Windows.

REPO="keith-hung/Cyberian"
VERSION="v0.3.2"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CACHE_DIR="$SCRIPT_DIR/.cache"
BIN="$CACHE_DIR/slip"

# Fast path: binary already cached
if [[ -x "$BIN" ]]; then
    exec "$BIN" "$@"
fi

# Detect platform
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)        ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)             echo "slip: unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# WSL uses the linux binary
if [[ "$OS" == "linux" ]] && grep -qi microsoft /proc/version 2>/dev/null; then
    OS="linux"
fi

# Supported platform check (native Windows uses slip-launcher.ps1)
case "${OS}_${ARCH}" in
    linux_amd64|linux_arm64|darwin_amd64|darwin_arm64) ;;
    *) echo "slip: unsupported platform: ${OS}_${ARCH}" >&2; exit 1 ;;
esac

# Download binary. All progress goes to stderr so the daemon's stdout stays
# clean (it must print only the ID).
URL="https://github.com/${REPO}/releases/download/${VERSION}/slip_${OS}_${ARCH}"
mkdir -p "$CACHE_DIR"

echo "Downloading slip ${VERSION} for ${OS}/${ARCH}..." >&2
if command -v curl &>/dev/null; then
    curl -fsSL "$URL" -o "$BIN"
elif command -v wget &>/dev/null; then
    wget -qO "$BIN" "$URL"
else
    echo "slip: neither curl nor wget found" >&2
    exit 1
fi

chmod +x "$BIN"
echo "Downloaded successfully." >&2

exec "$BIN" "$@"
