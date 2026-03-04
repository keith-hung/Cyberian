#!/usr/bin/env bash
set -euo pipefail

# Cross-compile timecard-cli, wedaka-cli, and azuredevops-cli for all supported platforms.
# Usage:
#   ./scripts/build.sh              # build with version "dev"
#   ./scripts/build.sh v0.2.0       # build with explicit version

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-dev}"
COMMIT="$(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || echo "none")"
BUILD_DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
DIST_DIR="$REPO_ROOT/dist"

PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

CLIS=(
    "timecard-cli"
    "wedaka-cli"
    "azuredevops-cli"
)

# Clean previous build
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

echo "Building version=${VERSION} commit=${COMMIT} date=${BUILD_DATE}"
echo "Output: ${DIST_DIR}/"
echo "---"

for cli in "${CLIS[@]}"; do
    MODULE="$(grep '^module' "$REPO_ROOT/$cli/go.mod" | awk '{print $2}')"
    LDFLAGS="-s -w \
        -X ${MODULE}/cmd.Version=${VERSION} \
        -X ${MODULE}/cmd.Commit=${COMMIT} \
        -X ${MODULE}/cmd.BuildDate=${BUILD_DATE}"

    for platform in "${PLATFORMS[@]}"; do
        GOOS="${platform%/*}"
        GOARCH="${platform#*/}"

        if [[ "$GOOS" == "windows" ]]; then
            OUTPUT="${DIST_DIR}/${cli}_windows_${GOARCH}.exe"
        else
            OUTPUT="${DIST_DIR}/${cli}_${GOOS}_${GOARCH}"
        fi

        echo "  ${cli} → ${GOOS}/${GOARCH}"
        (cd "$REPO_ROOT/$cli" && CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
            go build -ldflags="$LDFLAGS" -o "$OUTPUT" .)
    done
done

echo "---"
echo "Done. Files:"
ls -lh "$DIST_DIR/"
