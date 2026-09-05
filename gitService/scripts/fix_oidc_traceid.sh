#!/usr/bin/env bash
# Ensure the OIDC traceId initializer is loaded in GitLab container.
# This injects data-traceId attribute into OIDC OmniAuth error flash alerts.
# Usage:
#   fix_oidc_traceid.sh              Check and apply fix
# Pre: GitLab container running (docker ps includes gitlab)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTAINER="${GITLAB_CONTAINER:-gitlab}"
INITIALIZER="/opt/gitlab/embedded/service/gitlab-rails/config/initializers/zzz_fix_oidc_traceid.rb"
SOURCE="$SCRIPT_DIR/../initializers/zzz_fix_oidc_traceid.rb"

echo "=== OIDC TraceId Injection Fix ==="

# ── 1. Check container is running ──────────────────────────────────────────
if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  echo "GitLab 容器 $CONTAINER 未运行，跳过。" >&2
  exit 0
fi

# ── 2. Check if source file exists ─────────────────────────────────────────
if [[ ! -f "$SOURCE" ]]; then
  echo "❌ 源文件不存在: $SOURCE" >&2
  exit 1
fi

# ── 3. Check if fix already applied ────────────────────────────────────────
EXISTING=$(docker exec "$CONTAINER" cat "$INITIALIZER" 2>/dev/null || echo "")
if echo "$EXISTING" | grep -q "OmniAuthTraceId"; then
  echo "✅ 修复已应用 — zzz_fix_oidc_traceid.rb 已加载"

  # Verify Puma has loaded the module (check runtime)
  if docker exec "$CONTAINER" gitlab-rails runner "puts defined?(OmniAuthTraceId) ? 'loaded' : 'not_loaded'" 2>/dev/null | grep -q "loaded"; then
    echo "✅ OmniAuthTraceId 模块已激活"
    exit 0
  else
    echo "⚠️  模块未激活，重启 Puma..."
  fi
else
  echo "→ 安装 OmniAuthTraceId 初始化器..."
  # Check if already bind-mounted (docker-compose handles this)
  if docker exec "$CONTAINER" test -f "$INITIALIZER" 2>/dev/null; then
    echo "  文件已存在但内容不同，更新中..."
    docker cp "$SOURCE" "$CONTAINER:$INITIALIZER"
  else
    echo "  初次安装..."
    docker cp "$SOURCE" "$CONTAINER:$INITIALIZER"
  fi
fi

# ── 4. Restart Puma to load the new initializer ────────────────────────────
echo "→ 重启 Puma 以加载新初始化器..."
docker exec "$CONTAINER" gitlab-ctl restart puma 2>&1 | while IFS= read -r line; do
  echo "  [gitlab-ctl] $line"
done

# ── 5. Wait for ready and verify ───────────────────────────────────────────
echo "→ 等待 GitLab Rails 就绪..."
MAX_WAIT=120
WAITED=0
while (( WAITED < MAX_WAIT )); do
  if docker exec "$CONTAINER" gitlab-rails runner "puts 'ready'" >/dev/null 2>&1; then
    echo "GitLab 就绪。"
    break
  fi
  sleep 5
  WAITED=$((WAITED + 5))
done

if docker exec "$CONTAINER" gitlab-rails runner "puts defined?(OmniAuthTraceId) ? 'loaded' : 'not_loaded'" 2>/dev/null | grep -q "loaded"; then
  echo "✅ OmniAuthTraceId 模块已激活"
else
  echo "⚠️  模块未激活 — 请检查 Rails 日志: docker exec $CONTAINER tail -100 /var/log/gitlab/puma/puma.stderr.log"
fi

echo "=== TraceId 注入修复完成 ==="
