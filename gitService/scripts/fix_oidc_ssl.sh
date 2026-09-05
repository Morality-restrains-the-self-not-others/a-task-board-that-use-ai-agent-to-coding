#!/usr/bin/env bash
# 确保 GitLab 容器中的 SWD.url_builder 设置为 URI::HTTP。
# 修复 openid_connect gem 强制 HTTP→HTTPS 导致的 SSL record layer failure。
# 用法:
#   fix_oidc_ssl.sh              检查并应用修复
# 前置: GitLab 容器正在运行（docker ps 包含 gitlab）
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTAINER="${GITLAB_CONTAINER:-gitlab}"
INITIALIZER="/opt/gitlab/embedded/service/gitlab-rails/config/initializers/zzz_fix_oidc_http.rb"

echo "=== OIDC SSL Protocol Fix ==="

# ── 1. Check container is running ──────────────────────────────────────────
if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  echo "GitLab 容器 $CONTAINER 未运行，跳过。" >&2
  exit 0
fi

# ── 2. Check if fix already applied ────────────────────────────────────────
EXISTING=$(docker exec "$CONTAINER" cat "$INITIALIZER" 2>/dev/null || echo "")
if echo "$EXISTING" | grep -q "SWD.url_builder = URI::HTTP"; then
  echo "✅ 修复已应用 — SWD.url_builder = URI::HTTP"
  exit 0
fi

# ── 3. Wait for GitLab to be ready ─────────────────────────────────────────
echo "等待 GitLab Rails 就绪..."
MAX_WAIT=300
WAITED=0
while (( WAITED < MAX_WAIT )); do
  if docker exec "$CONTAINER" gitlab-rails runner "puts 'ready'" >/dev/null 2>&1; then
    echo "GitLab Rails 就绪。"
    break
  fi
  sleep 5
  WAITED=$((WAITED + 5))
done
if (( WAITED >= MAX_WAIT )); then
  echo "GitLab Rails 未能在 ${MAX_WAIT}s 内就绪，跳过修复。" >&2
  exit 0
fi

# ── 4. Inject initializer ──────────────────────────────────────────────────
echo "注入 OIDC protocol fix initializer..."
docker exec "$CONTAINER" bash -c "cat > $INITIALIZER <<'RUBY'
require \"swd\"
SWD.url_builder = URI::HTTP
RUBY"

# Verify injection
VERIFY=$(docker exec "$CONTAINER" cat "$INITIALIZER" 2>/dev/null || echo "")
if ! echo "$VERIFY" | grep -q "SWD.url_builder = URI::HTTP"; then
  echo "❌ 注入验证失败" >&2
  exit 1
fi
echo "✅ Initializer 已注入: $INITIALIZER"

# ── 5. Reconfigure GitLab ──────────────────────────────────────────────────
echo "执行 gitlab-ctl reconfigure（约需 1-3 分钟）..."
if docker exec "$CONTAINER" gitlab-ctl reconfigure 2>&1 | while IFS= read -r line; do
  echo "  [gitlab-ctl] $line"
done; then
  echo "✅ reconfigure 完成"

  # ── 5b. Restart Puma to load the new initializer ────────────────────────
  # Note: reconfigure may skip Puma restart if no Chef resources changed.
  # We explicitly restart Puma so SWD.url_builder takes effect in the web process.
  echo "重启 Puma 以加载 initializer..."
  docker exec "$CONTAINER" gitlab-ctl restart puma 2>&1 \
    || echo "提示: Puma 重启失败，可能需要手动重启" >&2

  # ── 6. Wait for GitLab to come back ─────────────────────────────────────
  echo "等待 GitLab 恢复..."
  WAITED=0
  while (( WAITED < MAX_WAIT )); do
    if docker exec "$CONTAINER" gitlab-rails runner "puts 'ready'" >/dev/null 2>&1; then
      echo "GitLab 已恢复。"
      break
    fi
    sleep 5
    WAITED=$((WAITED + 5))
  done

  # ── 7. Verify fix is loaded ─────────────────────────────────────────────
  echo "验证 SWD.url_builder 已生效..."
  SWD_CHECK=$(docker exec "$CONTAINER" gitlab-rails runner "
    require 'swd'
    puts SWD.url_builder
  " 2>&1 || echo "CHECK_FAILED")

  if echo "$SWD_CHECK" | grep -q "URI::HTTP"; then
    echo "✅ SWD.url_builder = URI::HTTP 已生效"
  else
    echo "⚠️  SWD.url_builder 验证返回: $SWD_CHECK"
    echo "   请手动检查: docker exec $CONTAINER gitlab-rails runner \"require 'swd'; puts SWD.url_builder\""
  fi
else
  echo "❌ gitlab-ctl reconfigure 失败" >&2
  exit 1
fi

echo "=== OIDC SSL Protocol Fix 完成 ==="
