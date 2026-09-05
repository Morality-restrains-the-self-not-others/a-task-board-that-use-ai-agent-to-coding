#!/usr/bin/env bash
# Shared Docker Desktop helpers for managed stacks (gitService, docker-infra, AiMonitor).
# Source this file; do not execute directly.

docker_helper_compose_cmd() {
  if docker compose version &>/dev/null; then
    echo "docker compose"
  elif command -v docker-compose &>/dev/null; then
    echo "docker-compose"
  else
    return 1
  fi
}

docker_helper_daemon_ready() {
  docker info >/dev/null 2>&1
}

docker_helper_desktop_memory_mib() {
  if ! command -v python3 &>/dev/null; then
    return 1
  fi
  python3 - <<'PY'
import json
import sys
from pathlib import Path

candidates = [
    Path.home() / "Library/Group Containers/group.com.docker/settings-store.json",
    Path.home() / "Library/Group Containers/group.com.docker/settings.json",
]
for path in candidates:
    if not path.is_file():
        continue
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        continue
    for key in ("MemoryMiB", "memoryMiB"):
        value = data.get(key)
        if isinstance(value, int) and value > 0:
            print(value)
            sys.exit(0)
sys.exit(1)
PY
}

docker_helper_try_start_desktop() {
  # 远程 Docker 方案：不再自动启动 Docker Desktop（避免与本机 SSH 隧道端口冲突）
  return 1
}

# Wait for daemon; on macOS optionally launch Docker Desktop first.
# Usage: docker_helper_ensure_daemon [max_wait_seconds]
docker_helper_ensure_daemon() {
  local max_wait="${1:-60}"
  local waited=0
  local launched=0

  while (( waited < max_wait )); do
    if docker_helper_daemon_ready; then
      return 0
    fi
    if (( launched == 0 )) && docker_helper_try_start_desktop; then
      launched=1
      max_wait=$(( max_wait > 120 ? max_wait : 120 ))
    fi
    sleep 1
    waited=$(( waited + 1 ))
  done
  return 1
}

# Warn when Docker Desktop VM memory is below the recommended threshold for GitLab + managed stacks.
# Usage: docker_helper_warn_gitlab_memory [min_mib]
docker_helper_warn_gitlab_memory() {
  local min_mib="${1:-6144}"
  local current_mib=""

  current_mib="$(docker_helper_desktop_memory_mib 2>/dev/null || true)"
  if [[ -z "$current_mib" ]]; then
    echo "提示: 无法读取 Docker Desktop 内存配置；GitLab 全栈建议至少分配 ${min_mib} MiB。" >&2
    return 0
  fi

  if (( current_mib < min_mib )); then
    echo "" >&2
    echo "警告: Docker Desktop 当前仅分配 ${current_mib} MiB 内存（建议 ≥ ${min_mib} MiB）。" >&2
    echo "GitLab 启动时易与其他容器（Kafka、AiMonitor 等）争抢内存，导致 Docker Engine 停止。" >&2
    echo "请在 Docker Desktop → Settings → Resources → Memory 调大后 Apply & Restart。" >&2
    echo "本次将使用精简版 GitLab 配置以降低占用，但仍可能不稳定。" >&2
    echo "" >&2
  fi
}
