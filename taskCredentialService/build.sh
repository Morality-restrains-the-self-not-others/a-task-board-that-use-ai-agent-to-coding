#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p bin
go build -buildvcs=false -o bin/taskCredentialService ./cmd/
echo "[task-credential-service] build complete: bin/taskCredentialService"
