#!/bin/bash
set -euo pipefail

# ============================================================
# Claude Agent — Build Script (Go)
# ============================================================
# Cross-compiles a single static binary.
#
# Usage:
#   ./build.sh              # Build for current platform
#   ./build.sh linux amd64  # Cross-compile for linux/amd64
# ============================================================

OS=${1:-linux}
ARCH=${2:-amd64}

BIN_DIR="$(cd "$(dirname "$0")" && pwd)/bin"
mkdir -p "$BIN_DIR"

BINARY="$BIN_DIR/claude-agent"
if [ "$OS" != "$(go env GOOS 2>/dev/null || echo linux)" ] || [ "$ARCH" != "$(go env GOARCH 2>/dev/null || echo amd64)" ]; then
    BINARY="$BIN_DIR/claude-agent-${OS}-${ARCH}"
fi

echo "=== Building claude-agent (Go) ==="
echo "  Target: $OS/$ARCH"
echo "  Output: $BINARY"

GONOSUMDB='*' GONOSUMCHECK='*' GOPROXY=off \
    GOOS="$OS" GOARCH="$ARCH" CGO_ENABLED=0 \
    go build -ldflags="-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo '0.2.0')" \
    -o "$BINARY" .

echo "  Size: $(du -h "$BINARY" | cut -f1)"
echo "=== Done ==="

# Also copy to repo root bin for runAll compatibility (no-copy if same)
REPO_BIN="$(cd "$(dirname "$0")" && pwd)/bin/claude-agent"
if [ "$(realpath "$BINARY" 2>/dev/null || echo "$BINARY")" != "$(realpath "$REPO_BIN" 2>/dev/null || echo "$REPO_BIN")" ]; then
    cp "$BINARY" "$REPO_BIN"
fi
