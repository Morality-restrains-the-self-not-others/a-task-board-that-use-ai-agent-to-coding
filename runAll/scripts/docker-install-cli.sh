#!/usr/bin/env bash
# 本机仅安装 docker CLI（无 Desktop）；清理已删除 Docker.app 的残留 symlink
set -euo pipefail

export PATH="/opt/homebrew/bin:/usr/local/bin:${PATH}"

if ! command -v brew &>/dev/null; then
  echo "错误: 需要 Homebrew（https://brew.sh）" >&2
  exit 1
fi

if ! command -v docker &>/dev/null; then
  echo "==> brew install docker"
  HOMEBREW_NO_AUTO_UPDATE=1 brew install docker
fi

echo "==> 清理 ~/.docker/cli-plugins 中指向 Docker.app 的断链"
find "${HOME}/.docker/cli-plugins" -type l ! -exec test -e {} \; -delete 2>/dev/null || true

if ! docker compose version &>/dev/null 2>&1; then
  echo "==> 安装 docker-compose（brew；远程 stack 经 SSH 执行，本机 compose 可选）"
  pkill -f "brew.rb install docker" 2>/dev/null || true
  rm -f "${HOME}/Library/Caches/Homebrew/downloads/"*docker-compose*.incomplete 2>/dev/null || true
  HOMEBREW_NO_AUTO_UPDATE=1 brew install docker-compose || {
    echo "警告: brew install docker-compose 失败；runAll 远程 infra 经 SSH 不依赖本机 compose。" >&2
  }
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"$ROOT/docker-context-init.sh"
"$ROOT/docker-use-remote.sh"

echo ""
echo "docker $(docker --version)"
docker compose version 2>/dev/null || echo "docker compose: 未安装（远程 SSH 模式可忽略）"
echo "context: $(docker context show 2>/dev/null || echo n/a)"
