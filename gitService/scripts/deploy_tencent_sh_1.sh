#!/usr/bin/env bash
# 在 Host sh（上海腾讯云）本机拉起独立 GitLab CE 实例 tencent-sh-1。
# 精简模式：HTTP :8014、SSH :2223（宿主机 sshd 已占 2222）。
# INFRA 机 9999 不要调用本脚本；用 scripts/runall_ssh_sh_gitlab.sh（ssh 到已部署的 compose 目录）。
set -euo pipefail
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GITSERVICE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
WORKSPACE_ROOT="$(cd "$GITSERVICE_DIR/.." && pwd)"

export GITSERVICE_CONF_APP="${GITSERVICE_CONF_APP:-git-service-tencent-sh-1}"
export GITLAB_OIDC_CLIENT_ID="${GITLAB_OIDC_CLIENT_ID:-gitlab-git-service-tencent-sh-1}"
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-gitservice-tencent-sh-1}"

mkdir -p /var/lib/daydaymoney/gitService-tencent-sh-1

cd "$GITSERVICE_DIR"
exec env GITSERVICE_CONF_APP="$GITSERVICE_CONF_APP" \
  GITLAB_OIDC_CLIENT_ID="$GITLAB_OIDC_CLIENT_ID" \
  COMPOSE_PROJECT_NAME="$COMPOSE_PROJECT_NAME" \
  "$GITSERVICE_DIR/run.sh"
