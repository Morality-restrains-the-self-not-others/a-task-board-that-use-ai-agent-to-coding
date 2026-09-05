#!/usr/bin/env bash
# ============================================================
# Claude Agent — All-in-one Setup (Go version)
# ============================================================
# Installs both claude-agent (Go static binary) and Claude Code (Node.js CLI)
# so the container / VM has everything it needs.
# ============================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  Claude Agent — All-in-one Setup (Go)${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# ---- Check prerequisites ----
echo -e "${BLUE}[1/4] Checking prerequisites...${NC}"

# Node.js
if ! command -v node &>/dev/null; then
    echo -e "${RED}Node.js not found. Please install Node.js >= 18.${NC}"
    echo "  curl -fsSL https://deb.nodesource.com/setup_20.x | bash -"
    echo "  apt-get install -y nodejs"
    exit 1
fi
NODE_VERSION=$(node --version)
echo -e "  ${GREEN}✓ Node.js ${NODE_VERSION}${NC}"

# Go
if ! command -v go &>/dev/null; then
    echo -e "${RED}Go not found. Please install Go >= 1.22.${NC}"
    echo "  https://go.dev/dl/"
    exit 1
fi
GO_VERSION=$(go version)
echo -e "  ${GREEN}✓ ${GO_VERSION}${NC}"

# ---- Install Claude Code (npm) ----
echo -e "${BLUE}[2/4] Installing Claude Code (npm)...${NC}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

if [ -f package.json ]; then
    npm install --omit=dev
    echo -e "  ${GREEN}✓ Claude Code installed${NC}"
else
    echo -e "  ${BLUE}Installing globally...${NC}"
    npm install -g @anthropic-ai/claude-code
    echo -e "  ${GREEN}✓ Claude Code installed globally${NC}"
fi

# ---- Build claude-agent (Go) ----
echo -e "${BLUE}[3/4] Building claude-agent (Go)...${NC}"
if [ -f build.sh ]; then
    bash build.sh
    echo -e "  ${GREEN}✓ claude-agent built${NC}"
else
    GONOSUMDB='*' GONOSUMCHECK='*' GOPROXY=off go build -ldflags="-s -w" -o bin/claude-agent .
    echo -e "  ${GREEN}✓ claude-agent built${NC}"
fi

# Symlink to PATH
BIN_DIR="$HOME/.local/bin"
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo -e "  ${BLUE}Note: $BIN_DIR is not in PATH. Add 'export PATH=\"\$HOME/.local/bin:\$PATH\"' to your shell rc.${NC}"
fi
mkdir -p "$BIN_DIR"
ln -sf "$SCRIPT_DIR/bin/claude-agent" "$BIN_DIR/claude-agent"
echo -e "  ${GREEN}✓ claude-agent linked to $BIN_DIR/claude-agent${NC}"

# ---- Verify ----
echo -e "${BLUE}[4/4] Verifying installation...${NC}"

if command -v claude &>/dev/null; then
    CLAUDE_VERSION=$(claude --version 2>&1 || echo "installed")
    echo -e "  ${GREEN}✓ claude CLI: ${CLAUDE_VERSION}${NC}"
else
    echo -e "  ${RED}✗ claude CLI not found in PATH${NC}"
    echo "  Try: export PATH=\"\$PATH:$SCRIPT_DIR/node_modules/.bin\""
fi

if "$SCRIPT_DIR/bin/claude-agent" --version &>/dev/null; then
    AGENT_VERSION=$("$SCRIPT_DIR/bin/claude-agent" --version 2>&1)
    echo -e "  ${GREEN}✓ claude-agent: ${AGENT_VERSION}${NC}"
else
    echo -e "  ${RED}✗ claude-agent binary not found${NC}"
fi

echo ""
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}  Setup complete!${NC}"
echo -e "${GREEN}============================================${NC}"
echo ""
echo "  Usage:"
echo "    claude-agent run \"your task\""
echo "    claude-agent interactive"
echo ""
echo "  Make sure ANTHROPIC_API_KEY is set:"
echo "    export ANTHROPIC_API_KEY=sk-ant-..."
