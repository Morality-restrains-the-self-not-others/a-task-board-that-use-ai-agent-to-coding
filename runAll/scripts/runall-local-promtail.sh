#!/usr/bin/env bash
# Mac: local Promtail overlay → remote Loki (A2 log shipping), desktop-linux context.
# Linux: promtail is integrated in AiMonitor/docker-compose.yaml (aimonitor-promtail);
#        this script only verifies Loki readiness and cleans legacy overlay containers.
set -euo pipefail
# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck source=docker-env.sh
source "$ROOT/runAll/scripts/docker-env.sh"

AIMONITOR="${ROOT}/AiMonitor"
COMPOSE_LOCAL="${AIMONITOR}/docker-compose.promtail-local.yaml"
CTX_LOCAL="${DOCKER_CTX_LOCAL:-desktop-linux}"

usage() {
  cat <<EOF
用法: $(basename "$0") <up|down|status|reset>

  up      启动本机 Promtail（tail \${RUNALL_LOG_ROOT} → 远程 Loki）
  down    停止本机 Promtail
  status  打印容器状态
  reset   停止并删除 promtail_local_data volume
  print-log-root  只打印将要 scrape 的宿主日志目录（不碰 Docker）

环境:
  RUNALL_LOG_ROOT     默认 <project_root>/logs（与 conf/runAll.yaml 一致）
  LOKI_PUSH_URL       默认从 conf/infra/docker-infra/config.yaml host 推导
  DOCKER_CTX_LOCAL    默认 desktop-linux
EOF
}

resolve_loki_push_url() {
  if [[ -n "${LOKI_PUSH_URL:-}" ]]; then
    echo "$LOKI_PUSH_URL"
    return 0
  fi
  # On Linux (non-Mac), Promtail runs in Docker alongside Loki on the same network
  if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "http://aimonitor-loki:3100/loki/api/v1/push"
    return 0
  fi
  local infra_yaml="${ROOT}/conf/infra/ai-monitor/config.yaml"
  if [[ ! -f "$infra_yaml" ]]; then
    echo "错误: 未设置 LOKI_PUSH_URL 且缺少 ${infra_yaml}" >&2
    return 1
  fi
  local host
  host="$(python3 -c "import yaml; d=yaml.safe_load(open('${infra_yaml}')); print((d.get('host') or '').strip())")"
  if [[ -z "$host" ]]; then
    echo "错误: conf/infra/ai-monitor/config.yaml 缺少 host" >&2
    return 1
  fi
  echo "http://${host}:3100/loki/api/v1/push"
}

resolve_runall_log_root() {
  if [[ -n "${RUNALL_LOG_ROOT:-}" ]]; then
    echo "$RUNALL_LOG_ROOT"
    return 0
  fi
  # Explicit DEPLOY_ROOT (tests / clone-run wrapper) wins over process discovery.
  if [[ -n "${DEPLOY_ROOT:-}" && -f "${DEPLOY_ROOT}/logs/task-task-service.log" ]]; then
    echo "${DEPLOY_ROOT}/logs"
    return 0
  fi
  # Live clone-run: orchestrator env is SSOT. ram-work conf often still says /tmp/ram-work/logs.
  if command -v pgrep >/dev/null 2>&1; then
    local pid live
    pid="$(pgrep -n -f '/bin/runAll' 2>/dev/null || true)"
    if [[ -n "${pid}" && -r "/proc/${pid}/environ" ]]; then
      live="$(tr '\0' '\n' < "/proc/${pid}/environ" | awk -F= '$1=="RUNALL_LOG_ROOT"{print $2; exit}')"
      if [[ -n "$live" && -d "$live" ]]; then
        echo "$live"
        return 0
      fi
    fi
  fi
  # Fallback when no live process and DEPLOY_ROOT unset (ADR-0052 live root).
  local deploy_logs="${DEPLOY_ROOT:-$HOME/bin/daydaymoney-deploy}/logs"
  if [[ -f "${deploy_logs}/task-task-service.log" ]]; then
    echo "$deploy_logs"
    return 0
  fi
  local runall_yaml="${ROOT}/conf/runAll.yaml"
  if [[ -f "$runall_yaml" ]] && command -v python3 >/dev/null 2>&1; then
    local raw
    raw="$(python3 -c "import yaml; d=yaml.safe_load(open('${runall_yaml}')); print((d.get('logging') or {}).get('file_root') or '')")"
    if [[ -n "$raw" ]]; then
      # 正确解析相对于项目根目录的相对路径
      # file_root: ../logs → ${ROOT}/../logs, file_root: logs → ${ROOT}/logs
      if [[ "$raw" = /* ]]; then
        echo "$raw"
      else
        # Resolve relative path against ROOT (project root, same as runAll CWD)
        echo "$(cd "${ROOT}" && realpath -m "$raw")"
      fi
      return 0
    fi
  fi
  echo "${ROOT}/logs"
}

is_linux_integrated() {
  [[ "$(uname -s)" != "Darwin" ]]
}

compose_local() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$COMPOSE_LOCAL" "$@"
  elif docker-compose version >/dev/null 2>&1; then
    docker-compose -f "$COMPOSE_LOCAL" "$@"
  else
    echo "错误: 需要 docker compose 或 docker-compose" >&2
    return 1
  fi
}

cleanup_linux_overlay_promtail() {
  if docker ps -a --filter "name=^aimonitor-promtail-local$" --format '{{.Names}}' 2>/dev/null | grep -q .; then
    echo "==> Linux: removing legacy aimonitor-promtail-local overlay container"
    docker rm -f aimonitor-promtail-local >/dev/null 2>&1 || true
  fi
}

wait_loki_ready() {
  local host="${1:-127.0.0.1}"
  local deadline=$((SECONDS + 90))
  while (( SECONDS < deadline )); do
    if curl -sf "http://${host}:3100/ready" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "警告: Loki http://${host}:3100/ready 在 90s 内未就绪（ai-monitor 可能仍在启动）" >&2
  return 0
}

ensure_local_context() {
  docker_env_require_cli
  docker_env_ensure_local_context
  CTX_LOCAL="${DOCKER_CTX_LOCAL:-desktop-linux}"
  docker context use "$CTX_LOCAL" >/dev/null
}

cmd_up() {
  ensure_local_context
  bash "${ROOT}/runAll/scripts/generate-promtail-config.sh"
  export RUNALL_LOG_ROOT="$(resolve_runall_log_root)"
  export LOKI_PUSH_URL="$(resolve_loki_push_url)"
  if [[ ! -d "$RUNALL_LOG_ROOT" ]]; then
    echo "警告: 日志目录 $RUNALL_LOG_ROOT 不存在。请先启动 runAll 产生 tee 日志。" >&2
    mkdir -p "$RUNALL_LOG_ROOT"
  fi

  if is_linux_integrated; then
    cleanup_linux_overlay_promtail
    echo "==> Linux integrated promtail: ensure aimonitor-promtail (compose up + reload)"
    echo "    tail ${RUNALL_LOG_ROOT} → ${LOKI_PUSH_URL}"
    # 可观测栈被 down/清空后容器可能不存在或 Exited：始终 compose up -d，再 restart 以加载新 scrape 配置。
    (
      cd "$AIMONITOR"
      export RUNALL_LOG_ROOT
      if docker compose version >/dev/null 2>&1; then
        docker compose up -d promtail
      else
        docker-compose up -d promtail
      fi
    )
    if docker ps -a --filter "name=^aimonitor-promtail$" --format '{{.Names}}' 2>/dev/null | grep -q .; then
      docker restart aimonitor-promtail >/dev/null 2>&1 || true
    fi
    docker ps --filter "name=^aimonitor-promtail$" --format '{{.Names}}\t{{.Status}}' 2>/dev/null \
      || echo "aimonitor-promtail\tfailed to start"
    wait_loki_ready "127.0.0.1"
    return 0
  fi

  echo "==> local promtail: tail ${RUNALL_LOG_ROOT} → ${LOKI_PUSH_URL}"
  compose_local up -d --force-recreate
  compose_local ps
}

cmd_down() {
  ensure_local_context
  if is_linux_integrated; then
    cleanup_linux_overlay_promtail
    # 与 ai-monitor stop 解耦：允许单独停采集而不拆掉 Grafana/Loki。
    if docker ps -a --filter "name=^aimonitor-promtail$" --format '{{.Names}}' 2>/dev/null | grep -q .; then
      echo "==> stopping aimonitor-promtail"
      docker stop aimonitor-promtail >/dev/null 2>&1 || true
    fi
    return 0
  fi
  export LOKI_PUSH_URL="$(resolve_loki_push_url)"
  compose_local down
}

cmd_status() {
  ensure_local_context
  if is_linux_integrated; then
    if docker ps -a --filter "name=^aimonitor-promtail$" --format '{{.Names}}\t{{.Status}}' 2>/dev/null | grep -q .; then
      docker ps -a --filter "name=^aimonitor-promtail$" --format '{{.Names}}\t{{.Status}}'
    else
      echo "aimonitor-promtail\tnot running"
    fi
    return 0
  fi
  if docker ps -a --filter "name=aimonitor-promtail-local" --format '{{.Names}}\t{{.Status}}' 2>/dev/null | grep -q .; then
    docker ps -a --filter "name=aimonitor-promtail-local" --format '{{.Names}}\t{{.Status}}'
  else
    echo "aimonitor-promtail-local\tnot running"
  fi
}

cmd_reset() {
  ensure_local_context
  export LOKI_PUSH_URL="$(resolve_loki_push_url)"
  compose_local down -v 2>/dev/null || true
  docker volume rm aimonitor_promtail_local_data 2>/dev/null \
    || docker volume rm "$(basename "$AIMONITOR")_promtail_local_data" 2>/dev/null \
    || true
  echo "local promtail data volume cleared"
}

main() {
  local cmd="${1:-}"
  case "$cmd" in
    up) cmd_up ;;
    down) cmd_down ;;
    status) cmd_status ;;
    reset) cmd_reset ;;
    print-log-root) resolve_runall_log_root ;;
    -h|--help|"") usage ;;
    *) echo "未知命令: $cmd" >&2; usage; exit 1 ;;
  esac
}

main "$@"
