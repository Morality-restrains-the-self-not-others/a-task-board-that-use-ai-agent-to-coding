#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=docker-env.sh
source "$ROOT/docker-env.sh"

docker_env_require_cli

ctx="$(docker_env_current_context)"
echo "当前 context: $ctx"
docker context ls

if docker_env_daemon_ready; then
  echo ""
  echo "Docker daemon: 可用"
  docker info --format '  Server: {{.Name}}  OS/{{.OSType}}  {{.DockerRootDir}}'
  docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' 2>/dev/null | head -20
else
  echo ""
  echo "Docker daemon: 不可用（请检查 context 或远程 SSH）" >&2
  exit 1
fi

if docker_env_is_remote_active; then
  echo ""
  if [[ -f "$DOCKER_TUNNEL_PID_FILE" ]] && kill -0 "$(cat "$DOCKER_TUNNEL_PID_FILE")" 2>/dev/null; then
    echo "SSH 隧道: 运行中 (pid $(cat "$DOCKER_TUNNEL_PID_FILE"))"
  else
    echo "SSH 隧道: 未运行 — 执行: $ROOT/docker-tunnel-remote.sh start"
  fi
fi
