#!/usr/bin/env bash
# 远程 / 本地 Docker 共享配置。被其他 docker-*.sh source，勿直接执行。
# shellcheck disable=SC2034

# Docker context 名称
DOCKER_CTX_REMOTE="zcpu-remote"
DOCKER_CTX_LOCAL="desktop-linux"

# SSH（与 ~/.ssh/config 中 Host zcpu 一致）
DOCKER_SSH_HOST="zcpu"
DOCKER_SSH_USER="ljy"
DOCKER_SSH_ADDR="10.2.150.119"

# 远程工作区根（AiMonitor bind mount / ensure-subrepo）
DOCKER_REMOTE_WORKSPACE="${DOCKER_REMOTE_WORKSPACE:-~/gitClone/ramDisk/ram-mount}"
readonly -a DOCKER_REMOTE_TUNNEL_PORTS_CORE=(
  6379    # redis
  9092    # kafka internal
  9093    # kafka bootstrap (port_config: localhost:9093)
  18080   # kafka-ui
)

readonly -a DOCKER_REMOTE_TUNNEL_PORTS_OBS=(
  3000    # grafana
  3100    # loki
  4317    # otel grpc
  4318    # otel http
)

readonly -a DOCKER_REMOTE_TUNNEL_PORTS_GIT=(
  8012    # gitlab http (port_config gitService.port)
  2222    # gitlab ssh (port_config gitService.sshPort)
)

readonly -a DOCKER_REMOTE_TUNNEL_PORTS=(
  "${DOCKER_REMOTE_TUNNEL_PORTS_CORE[@]}"
  "${DOCKER_REMOTE_TUNNEL_PORTS_OBS[@]}"
  "${DOCKER_REMOTE_TUNNEL_PORTS_GIT[@]}"
)

DOCKER_TUNNEL_PID_FILE="${TMPDIR:-/tmp}/ram-mount-docker-tunnel.pid"

docker_env_repo_root() {
  local src="${BASH_SOURCE[1]:-${BASH_SOURCE[0]}}"
  cd "$(dirname "$src")/.." && pwd
}

docker_env_require_cli() {
  if ! command -v docker &>/dev/null; then
    echo "错误: 未找到 docker 命令。Mac 可执行: brew install docker docker-compose" >&2
    return 1
  fi
}

docker_env_current_context() {
  docker context show 2>/dev/null || echo "unknown"
}

docker_env_is_remote_active() {
  [[ "$(docker_env_current_context)" == "$DOCKER_CTX_REMOTE" ]]
}

docker_env_reset_ssh_mux() {
  # 关闭 SSH 复用连接，避免旧会话缺少 docker 组权限
  ssh -O exit "$DOCKER_SSH_HOST" 2>/dev/null || true
}

docker_env_use_context() {
  local name="$1"
  docker context use "$name"
}

docker_env_ensure_remote_context() {
  if docker context inspect "$DOCKER_CTX_REMOTE" &>/dev/null; then
    return 0
  fi
  echo "创建远程 context: $DOCKER_CTX_REMOTE → ssh://$DOCKER_SSH_HOST"
  docker context create "$DOCKER_CTX_REMOTE" \
    --description "${DOCKER_SSH_USER}@${DOCKER_SSH_ADDR} 远程 Docker" \
    --docker "host=ssh://${DOCKER_SSH_HOST}"
}

docker_env_ensure_local_context() {
  if docker context inspect "$DOCKER_CTX_LOCAL" &>/dev/null; then
    return 0
  fi
  if [[ "$(uname -s)" == "Darwin" ]] && [[ -S "${HOME}/.docker/run/docker.sock" ]]; then
    echo "创建本地 context: $DOCKER_CTX_LOCAL → Docker Desktop"
    docker context create "$DOCKER_CTX_LOCAL" \
      --description "Docker Desktop" \
      --docker "host=unix://${HOME}/.docker/run/docker.sock"
    return 0
  fi
  # Linux fallback: use default Docker context if available
  if [[ -S /var/run/docker.sock ]] && docker info >/dev/null 2>&1; then
    echo "Linux 环境: 使用 default Docker context（无 desktop-linux）"
    DOCKER_CTX_LOCAL="default"
    return 0
  fi
  echo "警告: 未找到 Docker Desktop socket，本地 context 可能不可用。" >&2
  return 1
}

docker_env_daemon_ready() {
  docker info >/dev/null 2>&1
}

docker_env_print_context_hint() {
  :
}
