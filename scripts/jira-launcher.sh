#!/usr/bin/env bash
set -euo pipefail

REPO="ankitpokhrel/jira-cli"
VERSION="v1.7.0"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CACHE_DIR="$SCRIPT_DIR/.cache"
BIN="$CACHE_DIR/jira"

# --- Download binary if not cached ---
if [[ ! -x "$BIN" ]]; then
    # Detect platform
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64)        ARCH="x86_64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *)             echo '{"success":false,"error":"Unsupported architecture: '"$ARCH"'"}' >&2; exit 1 ;;
    esac

    # WSL uses linux binary
    if [[ "$OS" == "linux" ]] && grep -qi microsoft /proc/version 2>/dev/null; then
        OS="linux"
    fi

    # Map OS name to jira-cli release naming convention
    case "$OS" in
        linux)  RELEASE_OS="linux" ;;
        darwin) RELEASE_OS="macOS" ;;
        *)      echo '{"success":false,"error":"Unsupported OS: '"$OS"'"}' >&2; exit 1 ;;
    esac

    # Supported platform check
    case "${OS}_${ARCH}" in
        linux_x86_64|linux_arm64|darwin_x86_64|darwin_arm64) ;;
        *) echo '{"success":false,"error":"Unsupported platform: '"${OS}_${ARCH}"'"}' >&2; exit 1 ;;
    esac

    # Download and extract binary
    VERSION_NUM="${VERSION#v}"
    ARCHIVE="jira_${VERSION_NUM}_${RELEASE_OS}_${ARCH}.tar.gz"
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"
    mkdir -p "$CACHE_DIR"

    echo "Downloading jira-cli ${VERSION} for ${RELEASE_OS}/${ARCH}..." >&2
    TMPFILE="$CACHE_DIR/${ARCHIVE}"
    if command -v curl &>/dev/null; then
        curl -fsSL "$URL" -o "$TMPFILE"
    elif command -v wget &>/dev/null; then
        wget -qO "$TMPFILE" "$URL"
    else
        echo '{"success":false,"error":"Neither curl nor wget found"}' >&2
        exit 1
    fi

    # Extract binary from archive and clean up
    # Archive structure: jira_<ver>_<os>_<arch>/bin/jira (2 levels deep)
    tar xzf "$TMPFILE" -C "$CACHE_DIR" --strip-components=2 --wildcards '*/bin/jira' 2>/dev/null \
        || tar xzf "$TMPFILE" -C "$CACHE_DIR" --strip-components=1 --wildcards '*/jira' 2>/dev/null \
        || { echo '{"success":false,"error":"Failed to extract jira binary from archive"}' >&2; rm -f "$TMPFILE"; exit 1; }
    rm -f "$TMPFILE"

    chmod +x "$BIN"
    echo "Downloaded successfully." >&2
fi

# --- Auto-init if env vars are set and no config exists ---
JIRA_CONFIG="${HOME}/.config/.jira/.config.yml"
if [[ ! -f "$JIRA_CONFIG" ]] && [[ -n "${JIRA_API_TOKEN:-}" ]] && [[ -n "${JIRA_SERVER_URL:-}" ]] && [[ -n "${JIRA_USER_EMAIL:-}" ]]; then
    INIT_ARGS=(--installation cloud --server "$JIRA_SERVER_URL" --login "$JIRA_USER_EMAIL" --force)
    [[ -n "${JIRA_PROJECT:-}" ]] && INIT_ARGS+=(--project "$JIRA_PROJECT")
    [[ -n "${JIRA_BOARD:-}" ]]   && INIT_ARGS+=(--board "$JIRA_BOARD")

    echo "Auto-initializing jira-cli config..." >&2
    "$BIN" init "${INIT_ARGS[@]}" >&2
fi

exec "$BIN" "$@"
