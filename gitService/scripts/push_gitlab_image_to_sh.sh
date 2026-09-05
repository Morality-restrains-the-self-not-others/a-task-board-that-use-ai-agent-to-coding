#!/usr/bin/env bash
# 将 GitLab CE 镜像推送到 Host sh（上海腾讯云）——绕过 sh 直连 Docker Hub 超时。
# OPT-20260818-041：把 ai.md 里手动的 save/load 管道固化为脚本，供升级 checklist 引用。
#
# 流程：INFRA docker save <image> → gzip → ssh sh 'gunzip | docker load'
#       然后（不在本脚本内）改 /opt/daydaymoney/gitservice-tencent-sh-1/.env 的
#       GITLAB_IMAGE=<image> 再 docker compose up -d（见 scripts/deploy_tencent_sh_1.sh）。
#
# 用法:
#   push_gitlab_image_to_sh.sh                # 推送 SSOT imageTag（git-service-tencent-sh-1）
#   push_gitlab_image_to_sh.sh --image gitlab/gitlab-ce:19.0.8-ce.0  # 分步升级覆盖
#   push_gitlab_image_to_sh.sh --check        # 只查 sh 是否已加载该镜像，不推送
#   GITSERVICE_SH_SSH_DRY_RUN=1 …             # 只打印管道命令，不执行
#
# 环境变量（与 runall_ssh_sh_gitlab.sh 对齐）:
#   GITSERVICE_SH_SSH_HOST (default sh)
#   GITSERVICE_SH_SSH_BIN (default ssh)
#   GITSERVICE_SH_SSH_CONNECT_TIMEOUT (default 15)
#   GITSERVICE_CONF_APP (default git-service-tencent-sh-1)
#   GITLAB_IMAGE_OVERRIDE (等价 --image)
set -euo pipefail
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GITSERVICE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
WORKSPACE_ROOT="$(cd "$GITSERVICE_DIR/.." && pwd)"

SSH_HOST="${GITSERVICE_SH_SSH_HOST:-sh}"
SSH_BIN="${GITSERVICE_SH_SSH_BIN:-ssh}"
CONNECT_TIMEOUT="${GITSERVICE_SH_SSH_CONNECT_TIMEOUT:-15}"
CONF_APP="${GITSERVICE_CONF_APP:-git-service-tencent-sh-1}"
IMAGE=""
ACTION="push"

usage() {
  echo "usage: $0 [--image gitlab/gitlab-ce:TAG] [--check] [--host HOST]" >&2
  echo "  --image  覆盖 SSOT imageTag（等价 GITLAB_IMAGE_OVERRIDE）" >&2
  echo "  --check  只查询 sh 是否已加载该镜像（不推送）" >&2
  echo "  --host   远程主机（default sh，等价 GITSERVICE_SH_SSH_HOST）" >&2
  exit 2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --image) IMAGE="${2:-}"; shift 2 ;;
    --check) ACTION="check"; shift ;;
    --host) SSH_HOST="${2:-}"; shift 2 ;;
    -h|--help) usage ;;
    *) usage ;;
  esac
done

# 解析镜像：--image / GITLAB_IMAGE_OVERRIDE → conf imageTag → 默认
resolve_image() {
  local override="${GITLAB_IMAGE_OVERRIDE:-$IMAGE}"
  if [[ -n "$override" ]]; then
    echo "$override"
    return
  fi
  local main_yaml="$WORKSPACE_ROOT/conf/infra/${CONF_APP}/config.yaml"
  if [[ ! -f "$main_yaml" ]]; then
    echo "error: SSOT 配置缺失: $main_yaml" >&2
    exit 2
  fi
  local config_json=""
  local conf_read="$WORKSPACE_ROOT/runAll/scripts/conf-read.py"
  if [[ -f "$conf_read" ]]; then
    config_json="$(python3 "$conf_read" "${CONF_APP}" --json 2>/dev/null || echo "")"
  fi
  python3 "$SCRIPT_DIR/load_gitservice_config.py" "$config_json" "$main_yaml" 2>/dev/null \
    | sed -n 's/^GITLAB_IMAGE=//p' | head -1
}

IMAGE="$(resolve_image)"
IMAGE="${IMAGE:-gitlab/gitlab-ce:19.2.4-ce.0}"

# 输入校验：远程主机与镜像名只允许安全字符，防注入 ssh / docker save 载荷。
case "$SSH_HOST" in
  *[!A-Za-z0-9._-]* | "")
    echo "error: GITSERVICE_SH_SSH_HOST 含非法字符: $SSH_HOST" >&2
    exit 2
    ;;
esac
case "$IMAGE" in
  *[!A-Za-z0-9.:/_-]* | "")
    echo "error: 镜像名含非法字符: $IMAGE" >&2
    exit 2
    ;;
esac

if [[ "$ACTION" == "check" ]]; then
  REMOTE_CMD="docker images --format '{{.Repository}}:{{.Tag}}' | grep -qx '${IMAGE}' && echo present || echo absent"
  echo "push_gitlab_image_to_sh: check host=${SSH_HOST} image=${IMAGE}" >&2
  if [[ "${GITSERVICE_SH_SSH_DRY_RUN:-}" == "1" ]]; then
    printf '%s -o BatchMode=yes -o ConnectTimeout=%s %s %q\n' \
      "$SSH_BIN" "$CONNECT_TIMEOUT" "$SSH_HOST" "$REMOTE_CMD"
    exit 0
  fi
  exec "$SSH_BIN" -o BatchMode=yes -o ConnectTimeout="$CONNECT_TIMEOUT" "$SSH_HOST" "$REMOTE_CMD"
fi

echo "push_gitlab_image_to_sh: push host=${SSH_HOST} image=${IMAGE}" >&2
if [[ "${GITSERVICE_SH_SSH_DRY_RUN:-}" == "1" ]]; then
  printf 'docker save %s | gzip -1 | %s -o BatchMode=yes -o ConnectTimeout=%s %s %q\n' \
    "$IMAGE" "$SSH_BIN" "$CONNECT_TIMEOUT" "$SSH_HOST" 'gunzip | docker load'
  exit 0
fi

# 3GB+ Omnibus 镜像；docker save 直接管道到远端 gunzip|docker load，不在 INFRA 落盘。
docker save "$IMAGE" | gzip -1 | "$SSH_BIN" -o BatchMode=yes -o ConnectTimeout="$CONNECT_TIMEOUT" "$SSH_HOST" 'gunzip | docker load'

echo "push_gitlab_image_to_sh: 已推送 ${IMAGE} → ${SSH_HOST}" >&2
echo "提示: 在 ${SSH_HOST} 上更新 /opt/daydaymoney/gitservice-tencent-sh-1/.env 的 GITLAB_IMAGE=${IMAGE} 后 docker compose up -d" >&2
