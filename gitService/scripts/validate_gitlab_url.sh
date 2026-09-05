#!/usr/bin/env bash
# 校验 GitLab 容器的 external_url 是否与 conf publicUrl/allowedHost 一致，
# 并在不一致时提供修复指引。
#
# 用法:
#   bash gitService/scripts/validate_gitlab_url.sh          # 仅检查
#   bash gitService/scripts/validate_gitlab_url.sh --fix    # 检查并自动修复
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GIT_SERVICE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
WORKSPACE_ROOT="$(cd "$GIT_SERVICE_DIR/.." && pwd)"
AUTO_FIX=false

if [[ "${1:-}" == "--fix" ]]; then
  AUTO_FIX=true
fi

CONTAINER_NAME="${GITLAB_CONTAINER:-gitlab}"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

msg_ok()  { echo -e "${GREEN}[OK]${NC} $*"; }
msg_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
msg_err() { echo -e "${RED}[ERR]${NC} $*"; }

# ── 1. 获取期望的 external_url（公网 SSOT，无 :8012）────────────────
get_expected_external_url() {
  local conf_read="$WORKSPACE_ROOT/runAll/scripts/conf-read.py"
  if [[ -f "$conf_read" ]] && command -v python3 &>/dev/null; then
    python3 "$conf_read" gitService --json 2>/dev/null | python3 -c "
import json, sys
from urllib.parse import urlparse
try:
    data = json.load(sys.stdin)
    url = (data.get('publicUrl') or data.get('allowedHost') or '').rstrip('/')
    if url:
        print(url)
except Exception:
    pass
"
  fi
}

# ── 2. 获取容器当前的 external_url ──────────────────────────────
get_current_external_url() {
  if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
    echo "CONTAINER_NOT_RUNNING"
    return
  fi
  docker exec "$CONTAINER_NAME" gitlab-rails runner \
    "puts Gitlab.config.gitlab.url.to_s.sub(%r{/+\z}, '')" 2>/dev/null || echo "ERROR"
}

# ── 主流程 ──────────────────────────────────────────────────────
echo "=== GitLab external_url 校验 ==="
echo ""

EXPECTED=$(get_expected_external_url)
if [[ -z "$EXPECTED" ]]; then
  msg_err "无法获取期望配置（conf-read.py 不可用或 publicUrl/allowedHost 缺失）"
  exit 2
fi
echo "期望 external_url:   $EXPECTED"

CURRENT=$(get_current_external_url)
if [[ "$CURRENT" == "CONTAINER_NOT_RUNNING" ]]; then
  msg_warn "GitLab 容器未运行，跳过校验。"
  exit 0
elif [[ "$CURRENT" == "ERROR" ]]; then
  msg_err "无法获取容器内 GitLab URL（容器可能正在初始化）"
  exit 3
fi
echo "当前 external_url:   $CURRENT"

# 规范化比较（去尾斜杠）
CURRENT_NORM="${CURRENT%/}"
EXPECTED_NORM="${EXPECTED%/}"

if [[ "$CURRENT_NORM" == "$EXPECTED_NORM" ]]; then
  msg_ok "external_url 一致: $CURRENT_NORM"
else
  msg_err "external_url 不一致！"
  echo "  当前: $CURRENT_NORM"
  echo "  期望: $EXPECTED_NORM"
  echo ""

  CURRENT_HOST=$(echo "$CURRENT_NORM" | sed -E 's|https?://([^/:]+)(:[0-9]+)?.*|\1|')
  case "$CURRENT_HOST" in
    localhost|127.*|::1|0.0.0.0)
      msg_warn "当前 external_url 使用 loopback 地址，页面显示的 clone URL 将不可从外部访问。"
      ;;
  esac
  if [[ "$CURRENT_NORM" == *":8012"* ]]; then
    msg_warn "当前 external_url 仍含 :8012；公网入口应为 HTTPS 无端口（边缘终止 TLS）。"
  fi

  if $AUTO_FIX; then
    echo "正在通过 gitService/run.sh 重建以应用 conf publicUrl…"
    (cd "$GIT_SERVICE_DIR" && bash run.sh) || {
      msg_err "run.sh 未能完成修复"
      exit 1
    }
    CURRENT=$(get_current_external_url)
    CURRENT_NORM="${CURRENT%/}"
    if [[ "$CURRENT_NORM" == "$EXPECTED_NORM" ]]; then
      msg_ok "修复后 external_url 已对齐: $CURRENT_NORM"
    else
      msg_err "修复后仍不一致: $CURRENT_NORM （期望 $EXPECTED_NORM）"
      exit 1
    fi
  else
    echo ""
    echo "修复方式:"
    echo "  bash $0 --fix"
    echo "  # 或: cd $GIT_SERVICE_DIR && bash run.sh"
    exit 1
  fi
fi

# ── 3. 额外检查：检测项目级别的 clone URL ──────────────────────
echo ""
echo "--- 项目 clone URL 抽查 ---"
if docker ps --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
  SAMPLE=$(docker exec "$CONTAINER_NAME" gitlab-rails runner \
    "p = Project.first; puts p ? p.http_url_to_repo : 'NO_PROJECTS'" 2>/dev/null || echo "ERROR")
  if [[ "$SAMPLE" != "ERROR" && "$SAMPLE" != "NO_PROJECTS" ]]; then
    EXPECTED_HOST=$(echo "$EXPECTED_NORM" | sed -E 's|https?://([^/:]+)(:[0-9]+)?.*|\1|')
    SAMPLE_HOST=$(echo "$SAMPLE" | sed -E 's|https?://([^/:]+).*|\1|')
    if [[ "$SAMPLE" == "$EXPECTED_NORM"* ]] || [[ "$SAMPLE_HOST" == "$EXPECTED_HOST" && "$SAMPLE" != *":8012"* ]]; then
      msg_ok "项目 clone URL 已对齐公网入口: $SAMPLE"
    else
      msg_warn "项目 clone URL 可能仍含旧地址"
      echo "  项目 URL 示例: $SAMPLE"
      echo "  期望前缀: $EXPECTED_NORM"
    fi
  else
    echo "（无项目或暂不可查）"
  fi
fi

echo ""
echo "校验完成。"
