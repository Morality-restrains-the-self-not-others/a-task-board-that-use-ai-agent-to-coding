#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=docker-env.sh
source "$ROOT/docker-env.sh"

docker_env_require_cli

docker_env_ensure_remote_context || true
docker_env_ensure_local_context || true

echo "Docker context 已就绪:"
docker context ls
