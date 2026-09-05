#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "$ROOT/../.." && pwd)"
SOURCE_LINE="source \"$REPO/runAll/scripts/docker-shell.sh\""

echo ""
echo "若尚未安装 docker CLI，请先执行: bash runAll/scripts/docker-install-cli.sh"
echo ""
echo "==> 初始化 Docker context"
export PATH="/opt/homebrew/bin:/usr/local/bin:${PATH}"
if command -v docker &>/dev/null; then
  "$ROOT/docker-context-init.sh"
  echo ""
  echo "==> 切换到远程 Docker（默认）"
  "$ROOT/docker-use-remote.sh"
else
  echo "跳过 context（未安装 docker CLI）"
fi

echo ""
echo "==> Shell 集成"
if grep -Fq "docker-shell.sh" "${HOME}/.zshrc" 2>/dev/null; then
  echo "~/.zshrc 已包含 docker-shell.sh，跳过"
else
  {
    echo ""
    echo "# ram-mount 远程 Docker 快捷命令 (dr/dl/ds/dt/dru)"
    echo "$SOURCE_LINE"
    echo "# export RAM_MOUNT_DOCKER_DEFAULT=remote  # 可选：新开终端默认远程"
  } >>"${HOME}/.zshrc"
  echo "已追加到 ~/.zshrc:"
  echo "  $SOURCE_LINE"
  echo "请执行: source ~/.zshrc"
fi

echo "==> 快捷命令"
cat <<'EOF'
  dr / dl / ds / dru / dt   Docker context 与隧道调试
EOF

echo ""
echo "Mac 仅需 docker CLI（brew install docker），无需 Docker Desktop；compose 在远程 CPU 执行。"
