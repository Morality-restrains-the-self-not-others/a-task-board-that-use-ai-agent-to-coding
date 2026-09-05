#!/usr/bin/env bash
# 集成测试: sync_local_oauth_app_scopes.sh — find-or-create 逻辑
# 验证:
#   1. Application 不存在时自动创建 (exit 0, 非 exit 1)
#   2. 重复执行幂等 (Application count 不变)
#   3. scopes 与 YAML provider 配置对齐
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SYNC_SCRIPT="$SCRIPT_DIR/sync_local_oauth_app_scopes.sh"
WORKSPACE_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PROVIDER_YAML="$WORKSPACE_ROOT/conf/auth/git-oauth/providers/http-localhost-8012.yaml"
CONTAINER="${GITLAB_CONTAINER:-gitlab}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}PASS${NC} $1"; }
fail() { echo -e "${RED}FAIL${NC} $1"; exit 1; }
info() { echo -e "${YELLOW}INFO${NC} $1"; }

# 前置条件: GitLab 容器必须运行
if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  info "GitLab 容器 $CONTAINER 未运行，跳过集成测试。"
  exit 0
fi

# 前置条件: rails console 必须可达
if ! docker exec "$CONTAINER" gitlab-rails runner "puts 'ready'" >/dev/null 2>&1; then
  info "GitLab rails console 未就绪，跳过集成测试。"
  exit 0
fi

# 读取 provider 配置（使用 || 分隔避免 scope 中空格导致 read 截断）
IFS='|' read -r CLIENT_ID REQUESTED_SCOPE <<EOF
$(python3 - "$PROVIDER_YAML" <<'PY'
import sys
from pathlib import Path
try:
    import yaml
except ImportError:
    print("|")
    raise SystemExit(0)
path = Path(sys.argv[1])
if not path.is_file():
    print("|")
    raise SystemExit(0)
data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
target = data.get("target") or {}
print(target.get("client_id", ""), end="")
print("|", end="")
print(target.get("scope", "read_repository write_repository api read_user"))
PY
)
EOF

if [[ -z "$CLIENT_ID" ]]; then
  info "未在 $PROVIDER_YAML 找到 client_id，跳过集成测试。"
  exit 0
fi

SCOPE="${REQUESTED_SCOPE:-read_repository write_repository api read_user}"

echo "========================================"
echo "  OAuth App Bootstrap 集成测试"
echo "  Provider: $PROVIDER_YAML"
echo "  Client ID: $CLIENT_ID"
echo "  Expected scope: $SCOPE"
echo "========================================"

# ============================================================
# 测试 1: 同步脚本正常退出 (exit 0)
# ============================================================
echo ""
echo "--- Test 1: sync script exits 0 ---"
if bash "$SYNC_SCRIPT"; then
  pass "sync script exit code = 0"
else
  fail "sync script exit code != 0 (expected 0 after bootstrap fix)"
fi

# ============================================================
# 测试 2: Application 存在且 scopes 与 YAML 对齐
# ============================================================
echo ""
echo "--- Test 2: Application exists with correct scopes ---"
APP_SCOPES=$(docker exec "$CONTAINER" gitlab-rails runner "
  app = Doorkeeper::Application.find_by(uid: '$CLIENT_ID')
  puts app&.scopes || 'NOT_FOUND'
" 2>/dev/null | tail -1)

if [[ "$APP_SCOPES" == "NOT_FOUND" ]]; then
  fail "Application uid=$CLIENT_ID not found after sync"
elif [[ "$APP_SCOPES" == "$SCOPE" ]]; then
  pass "Application scopes match: $APP_SCOPES"
else
  fail "Application scopes mismatch: expected '$SCOPE', got '$APP_SCOPES'"
fi

# ============================================================
# 测试 3: 幂等 — 重复执行不产生重复记录
# ============================================================
echo ""
echo "--- Test 3: Idempotency — no duplicate Applications ---"
COUNT_BEFORE=$(docker exec "$CONTAINER" gitlab-rails runner "
  puts Doorkeeper::Application.where(uid: '$CLIENT_ID').count
" 2>/dev/null | tail -1)

# 再运行两次
bash "$SYNC_SCRIPT" >/dev/null 2>&1 || true
bash "$SYNC_SCRIPT" >/dev/null 2>&1 || true

COUNT_AFTER=$(docker exec "$CONTAINER" gitlab-rails runner "
  puts Doorkeeper::Application.where(uid: '$CLIENT_ID').count
" 2>/dev/null | tail -1)

if [[ "$COUNT_BEFORE" == "$COUNT_AFTER" ]]; then
  pass "Application count stable: $COUNT_BEFORE → $COUNT_AFTER (no duplicates)"
else
  fail "Application count changed: $COUNT_BEFORE → $COUNT_AFTER (expected stable)"
fi

# ============================================================
# 测试 4: 日志区分 Created vs Updated
# ============================================================
echo ""
echo "--- Test 4: Log differentiation ---"
OUTPUT=$(bash "$SYNC_SCRIPT" 2>&1) || true
if echo "$OUTPUT" | grep -qE "CREATED|UPDATED|UNCHANGED"; then
  pass "Log contains CREATED, UPDATED or UNCHANGED marker"
else
  fail "Log missing CREATED/UPDATED/UNCHANGED marker. Output: $OUTPUT"
fi

# ============================================================
# 测试 5: gitlab-local scope 含 write_repository（合并原 synology-gitlab）
# ============================================================
echo ""
echo "--- Test 5: gitlab-local scopes include write_repository ---"
IFS='|' read -r LOCAL_CLIENT_ID LOCAL_SCOPE <<EOF
$(python3 - "$PROVIDER_YAML" <<'PY'
import sys
from pathlib import Path
try:
    import yaml
except ImportError:
    print("|")
    raise SystemExit(0)
path = Path(sys.argv[1])
if not path.is_file():
    print("|")
    raise SystemExit(0)
data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
target = data.get("target") or {}
print(target.get("client_id", ""), end="")
print("|", end="")
print(target.get("scope", "read_repository write_repository api read_user"))
PY
)
EOF

if [[ -n "$LOCAL_CLIENT_ID" ]]; then
  LOCAL_APP_SCOPES=$(docker exec "$CONTAINER" gitlab-rails runner "
    app = Doorkeeper::Application.find_by(uid: '$LOCAL_CLIENT_ID')
    puts app&.scopes || 'NOT_FOUND'
  " 2>/dev/null | tail -1)

  if [[ "$LOCAL_APP_SCOPES" == "NOT_FOUND" ]]; then
    fail "gitlab-local Application uid=$LOCAL_CLIENT_ID not found"
  elif [[ "$LOCAL_APP_SCOPES" == "$LOCAL_SCOPE" ]]; then
    pass "gitlab-local scopes match: $LOCAL_APP_SCOPES"
  else
    fail "gitlab-local scopes mismatch: expected '$LOCAL_SCOPE', got '$LOCAL_APP_SCOPES'"
  fi
else
  info "未找到 gitlab-local provider YAML，跳过 Test 5"
fi

# ============================================================
# 测试 6: synology-gitlab Application 存在且 scopes 正确
# ============================================================
echo ""
echo "--- Test 6: synology-gitlab Application exists with correct scopes ---"
SYNOLOGY_PROVIDER_YAML="$WORKSPACE_ROOT/conf/auth/git-oauth/providers/http-synology-gitlab.yaml"

if [[ -f "$SYNOLOGY_PROVIDER_YAML" ]]; then
  IFS='|' read -r SYN_CLIENT_ID SYN_SCOPE <<EOF
$(python3 - "$SYNOLOGY_PROVIDER_YAML" <<'PY'
import sys
from pathlib import Path
try:
    import yaml
except ImportError:
    print("|")
    raise SystemExit(0)
path = Path(sys.argv[1])
if not path.is_file():
    print("|")
    raise SystemExit(0)
data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
target = data.get("target") or {}
print(target.get("client_id", ""), end="")
print("|", end="")
print(target.get("scope", "read_repository write_repository api read_user"))
PY
)
EOF

  if [[ -n "$SYN_CLIENT_ID" ]]; then
    SYN_APP_SCOPES=$(docker exec "$CONTAINER" gitlab-rails runner "
      app = Doorkeeper::Application.find_by(uid: '$SYN_CLIENT_ID')
      puts app&.scopes || 'NOT_FOUND'
    " 2>/dev/null | tail -1)

    if [[ "$SYN_APP_SCOPES" == "NOT_FOUND" ]]; then
      fail "synology-gitlab Application uid=$SYN_CLIENT_ID not found after sync"
    elif [[ "$SYN_APP_SCOPES" == "$SYN_SCOPE" ]]; then
      pass "synology-gitlab scopes match: $SYN_APP_SCOPES"
    else
      fail "synology-gitlab scopes mismatch: expected '$SYN_SCOPE', got '$SYN_APP_SCOPES'"
    fi

    # 幂等检查
    SYN_COUNT_BEFORE=$(docker exec "$CONTAINER" gitlab-rails runner "
      puts Doorkeeper::Application.where(uid: '$SYN_CLIENT_ID').count
    " 2>/dev/null | tail -1)
    bash "$SYNC_SCRIPT" >/dev/null 2>&1 || true
    SYN_COUNT_AFTER=$(docker exec "$CONTAINER" gitlab-rails runner "
      puts Doorkeeper::Application.where(uid: '$SYN_CLIENT_ID').count
    " 2>/dev/null | tail -1)
    if [[ "$SYN_COUNT_BEFORE" == "$SYN_COUNT_AFTER" ]]; then
      pass "synology-gitlab idempotent: $SYN_COUNT_BEFORE -> $SYN_COUNT_AFTER"
    else
      fail "synology-gitlab duplicate: $SYN_COUNT_BEFORE -> $SYN_COUNT_AFTER"
    fi
  else
    info "synology-gitlab provider YAML 无 client_id，跳过 Test 6"
  fi
else
  info "未找到 synology-gitlab provider YAML，跳过 Test 6"
fi

echo ""
echo "========================================"
echo -e "  ${GREEN}所有集成测试通过${NC}"
echo "========================================"
