#!/usr/bin/env bash
set -euo pipefail

REPO="keith-hung/timecard-cli"
VERSION="v0.1.0"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CACHE_DIR="$SCRIPT_DIR/.cache"
BIN="$CACHE_DIR/timecard-cli"

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
    *)             echo '{"success":false,"error":"Unsupported architecture: '"$ARCH"'"}' >&2; exit 1 ;;
esac

# WSL uses linux binary
if [[ "$OS" == "linux" ]] && grep -qi microsoft /proc/version 2>/dev/null; then
    OS="linux"
fi

# Supported platform check
case "${OS}_${ARCH}" in
    linux_amd64|linux_arm64|darwin_amd64|darwin_arm64) ;;
    *) echo '{"success":false,"error":"Unsupported platform: '"${OS}_${ARCH}"'"}' >&2; exit 1 ;;
esac

# Download binary
URL="https://github.com/${REPO}/releases/download/${VERSION}/timecard-cli_${OS}_${ARCH}"
mkdir -p "$CACHE_DIR"

echo "Downloading timecard-cli ${VERSION} for ${OS}/${ARCH}..." >&2
if command -v curl &>/dev/null; then
    curl -fsSL "$URL" -o "$BIN"
elif command -v wget &>/dev/null; then
    wget -qO "$BIN" "$URL"
else
    echo '{"success":false,"error":"Neither curl nor wget found"}' >&2
    exit 1
fi

chmod +x "$BIN"
echo "Downloaded successfully." >&2

exec "$BIN" "$@"
