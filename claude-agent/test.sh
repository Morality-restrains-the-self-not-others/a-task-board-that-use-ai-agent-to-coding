#!/bin/bash
set -euo pipefail

# ============================================================
# Claude Agent — Test Script (Go)
# ============================================================
echo "=== Running claude-agent tests ==="
GONOSUMDB='*' GONOSUMCHECK='*' GOPROXY=off go test ./src/... -v -count=1 "$@"
echo "=== All tests passed ==="
