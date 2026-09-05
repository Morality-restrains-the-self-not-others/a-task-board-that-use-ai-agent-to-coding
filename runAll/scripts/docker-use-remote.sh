#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=docker-env.sh
source "$ROOT/docker-env.sh"

docker_env_require_cli
docker_env_reset_ssh_mux
docker_env_ensure_remote_context
docker_env_use_context "$DOCKER_CTX_REMOTE"
echo "已切换到远程 Docker ${DOCKER_SSH_USER}@${DOCKER_SSH_ADDR} (context: $DOCKER_CTX_REMOTE)"
docker context ls
docker_env_print_context_hint
