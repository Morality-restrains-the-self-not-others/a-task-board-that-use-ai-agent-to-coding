#!/usr/bin/env bash
# 卸载本机 Docker Desktop（保留 brew docker CLI + zcpu-remote context）
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=docker-env.sh
source "$ROOT/docker-env.sh"

echo "==> 停止 Docker Desktop 进程"
osascript -e 'quit app "Docker"' 2>/dev/null || true
sleep 2
pkill -x "Docker Desktop" 2>/dev/null || true
pkill -x "com.docker.backend" 2>/dev/null || true

if [[ -d /Applications/Docker.app ]]; then
  echo "==> 运行 Docker Desktop 卸载程序（若存在）"
  if [[ -x /Applications/Docker.app/Contents/MacOS/uninstall ]]; then
    /Applications/Docker.app/Contents/MacOS/uninstall || true
  fi
  echo "==> 删除 /Applications/Docker.app"
  rm -rf /Applications/Docker.app
else
  echo "Docker.app 未安装，跳过删除"
fi

echo "==> 切换到远程 Docker context"
if ! command -v docker &>/dev/null; then
  if command -v brew &>/dev/null; then
    echo "未找到 docker CLI，正在 brew install docker …"
    HOMEBREW_NO_AUTO_UPDATE=1 brew install docker
  else
    echo "错误: 请先安装 docker CLI（brew install docker）" >&2
    exit 1
  fi
fi
export PATH="/opt/homebrew/bin:/usr/local/bin:${PATH}"
# 卸载 Desktop 后旧 symlink 可能指向已删除的 Docker.app
if [[ -L /usr/local/bin/docker ]] && [[ ! -e /usr/local/bin/docker ]]; then
  rm -f /usr/local/bin/docker /usr/local/bin/docker-compose 2>/dev/null || true
fi
docker_env_require_cli
docker_env_ensure_remote_context
docker_env_use_context "$DOCKER_CTX_REMOTE"

echo ""
echo "完成。本机保留 docker CLI；基础设施与 GitLab 均在远程 ZCPU 运行。"
echo "验证: docker context show && docker info"
echo "启动: cd runAll && ./run.sh   # 常驻入口（setsid 独立会话）；调试前台用 ./bin/runAll --config ../conf/runAll.yaml"
