#!/usr/bin/env bash
# OAuth smoke: health, OpenAPI, bridge secret, start-from-gateway, access-for-user.
set -euo pipefail
BASE="${GITOAUTH_BASE:-http://127.0.0.1:8002}"
SECRET="${GITOAUTH_BRIDGE_JWT_SECRET:-${TASK2APP_SSO_JWT_SECRET:-task2app-local-sso-bridge-dev-do-not-use-in-prod}}"
UID_GH="${SMOKE_USER_ID:-1}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DB="${GITOAUTH_DATABASE_PATH:-$ROOT/../db/git-oauth/git-oauth.sqlite3}"

# 存储 MySQL 化（db/registry.yaml driver: mysql）：优先 mysql 客户端，回退 legacy sqlite3。
# 解析顺序：GITOAUTH_MYSQL_DSN → db/registry.yaml mysql 块；密码含特殊字符时请用 env 覆盖。
REG="$ROOT/../db/registry.yaml"
MYSQL_ARGS=""
if [ -n "${GITOAUTH_MYSQL_DSN:-}" ]; then
  MYSQL_ARGS=$(python3 -c "import re,sys; m=re.match(r'^([^:]+):([^@]*)@tcp\(([^:]+):(\d+)\)/(\w+)', sys.argv[1]); print(f\"-h {m.group(3)} -P {m.group(4)} -u {m.group(1)} -p{m.group(2)} {m.group(5)}\") if m else ''" "$GITOAUTH_MYSQL_DSN")
elif [ -f "$REG" ]; then
  H=$(sed -n 's/^  host: *//p' "$REG" | head -1)
  P=$(sed -n 's/^  port: *//p' "$REG" | head -1)
  U=$(sed -n 's/^  user: *//p' "$REG" | head -1)
  PW=$(sed -n 's/^  password: *//p' "$REG" | head -1)
  D=$(awk '/^  git-oauth:/{f=1} f && /^    database: /{print $2; exit}' "$REG")
  [ -n "$H" ] && [ -n "$D" ] && MYSQL_ARGS="-h $H -P $P -u $U -p$PW $D"
fi
q() { # 查询 helper：MySQL 优先，SQLite 回退（仅当 legacy 文件存在）
  if [ -n "$MYSQL_ARGS" ]; then mysql -N -B $MYSQL_ARGS -e "$1"; else sqlite3 "$DB" "$1"; fi
}
FAIL=0

echo "==> health"
curl -sf "$BASE/api/health/" | tee /tmp/smoke_health.json >/dev/null
python3 -c "import json; d=json.load(open('/tmp/smoke_health.json')); assert d.get('ok') is True, d; print('health ok')"

echo "==> openapi schemas"
curl -sf "$BASE/api/swagger/" | tee /tmp/smoke_swagger.json >/dev/null
python3 -c "import json; d=json.load(open('/tmp/smoke_swagger.json')); assert 'AccessForUserRequest' in d.get('components',{}).get('schemas',{}), list(d.keys()); print('openapi schemas ok paths=', len(d.get('paths',{})))"

echo "==> bridge secret rejected without header"
code=$(curl -s -o /tmp/smoke_unauth.json -w '%{http_code}' -X POST "$BASE/api/internal/github/oauth/access-for-user/" \
  -H 'Content-Type: application/json' -d '{"user_id":1}')
if [[ "$code" != "401" ]]; then
  echo "want 401 got $code"; cat /tmp/smoke_unauth.json; FAIL=1
else
  echo "unauthorized=$code ok"
fi

echo "==> GitHub start-from-gateway authorize_url"
code=$(curl -s -o /tmp/smoke_gh_start.json -w '%{http_code}' \
  "$BASE/api/accounts/github/oauth/start-from-gateway/?next=/profile/git-site-oauth/" \
  -H "X-User-Id: $UID_GH")
if ! python3 -c "import json; d=json.load(open('/tmp/smoke_gh_start.json')); assert int('$code')==200 and 'github.com/login/oauth/authorize' in str(d.get('authorize_url','')), d"; then
  echo "github start failed"; FAIL=1
else
  echo "github authorize_url ok"
fi

echo "==> GitLab start-from-gateway"
code=$(curl -s -o /tmp/smoke_gl_start.json -w '%{http_code}' \
  "$BASE/api/accounts/gitlab/oauth/start-from-gateway/?service_provider=daydaymoney-gitlab&next=/profile/git-site-oauth/" \
  -H "X-User-Id: $UID_GH")
if ! python3 -c "import json; d=json.load(open('/tmp/smoke_gl_start.json')); assert int('$code')==200 and 'oauth/authorize' in str(d.get('authorize_url','')), d"; then
  echo "daydaymoney failed; trying gitlab-local"
  code=$(curl -s -o /tmp/smoke_gl_start.json -w '%{http_code}' \
    "$BASE/api/accounts/gitlab/oauth/start-from-gateway/?service_provider=gitlab-local&allowed_host=http://127.0.0.1:8012&next=/profile/git-site-oauth/" \
    -H "X-User-Id: $UID_GH")
  if ! python3 -c "import json; d=json.load(open('/tmp/smoke_gl_start.json')); assert int('$code')==200 and 'oauth/authorize' in str(d.get('authorize_url','')), d"; then
    echo "gitlab start failed"; FAIL=1
  else
    echo "gitlab-local authorize_url ok"
  fi
else
  echo "gitlab daydaymoney authorize_url ok"
fi

echo "==> GitLab access-for-user (stored active credential)"
row=$(q "SELECT CONCAT(task2app_user_id,'|',provider) FROM git_oauth_appusercredential WHERE provider LIKE 'gitlab:%' AND bind_status='active' ORDER BY updated_at DESC LIMIT 1;" || true)
if [[ -n "$row" ]]; then
  GL_UID="${row%%|*}"
  GL_PK="${row#*|}"
  code=$(curl -s -o /tmp/smoke_gl_access.json -w '%{http_code}' -X POST \
    "$BASE/api/internal/gitlab/oauth/access-for-user/" \
    -H "Content-Type: application/json" \
    -H "X-GitOauth-Bridge-Secret: $SECRET" \
    -d "{\"user_id\": $GL_UID, \"provider_key\": \"$GL_PK\"}")
  if ! python3 -c "import json; d=json.load(open('/tmp/smoke_gl_access.json')); assert int('$code')==200 and d.get('access_token'), d"; then
    echo "gitlab access failed code=$code"; cat /tmp/smoke_gl_access.json; FAIL=1
  else
    python3 -c "import json; d=json.load(open('/tmp/smoke_gl_access.json')); print('gitlab access-for-user ok', 'token_len', len(d['access_token']), 'provider', '$GL_PK')"
  fi
else
  echo "WARN: no active gitlab credential"; FAIL=1
fi

echo "==> GitHub access-for-user if credential exists"
row=$(q "SELECT CONCAT(task2app_user_id,'|',provider) FROM git_oauth_appusercredential WHERE (provider='github' OR provider LIKE 'github:%') AND bind_status='active' ORDER BY updated_at DESC LIMIT 1;" || true)
if [[ -n "$row" ]]; then
  GH_UID="${row%%|*}"
  code=$(curl -s -o /tmp/smoke_gh_access.json -w '%{http_code}' -X POST \
    "$BASE/api/internal/github/oauth/access-for-user/" \
    -H "Content-Type: application/json" \
    -H "X-GitOauth-Bridge-Secret: $SECRET" \
    -d "{\"user_id\": $GH_UID, \"provider_key\": \"github:github-official\"}")
  if ! python3 -c "import json; d=json.load(open('/tmp/smoke_gh_access.json')); assert int('$code')==200 and d.get('access_token'), d"; then
    echo "github access failed"; FAIL=1
  else
    python3 -c "import json; d=json.load(open('/tmp/smoke_gh_access.json')); print('github access-for-user ok token_len', len(d['access_token']))"
  fi
else
  echo "INFO: no active github credential in DB (start-from-gateway covered GitHub authorize URL)"
fi

if [[ "$FAIL" -ne 0 ]]; then
  echo "SMOKE FAILED"
  exit 1
fi
echo "SMOKE PASSED"
