#!/usr/bin/env bash
# 回归：GitLab OmniAuth redirect_uri 必须与 taskAuth bootstrap 白名单一致（公网 HTTPS、无 :8012）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
TASKAUTH="$ROOT/conf/auth/task-auth/config.yaml"
BASE="$ROOT/conf/base.yaml"

RESOLVED="$(python3 - "$TASKAUTH" "$BASE" <<'PY'
import os, re, sys
from pathlib import Path
import yaml

task_auth = Path(sys.argv[1])
base_path = Path(sys.argv[2])

def expand_env_default(value: str) -> str:
    return re.sub(
        r"\$\{(\w+):-([^}]*)\}",
        lambda m: os.environ.get(m.group(1), m.group(2) or ""),
        value,
    )

base = yaml.safe_load(base_path.read_text(encoding="utf-8")) or {}
scheme = expand_env_default(str(base.get("scheme") or "https")).strip().lower().rstrip(":/") or "https"
domain = expand_env_default(str(base.get("baseDomain") or ""))
m = {"scheme": scheme, "baseDomain": domain}
for k, tmpl in (base.get("subdomains") or {}).items():
    m[f"subdomains.{k}"] = str(tmpl).replace("${scheme}", scheme).replace("${baseDomain}", domain)

data = yaml.safe_load(task_auth.read_text(encoding="utf-8")) or {}
for client in (data.get("oidc") or {}).get("bootstrapClients") or []:
    if str(client.get("clientId") or "") == "gitlab-git-service":
        uri = str(client.get("redirectUri") or "")
        for k, v in m.items():
            uri = uri.replace("${" + k + "}", v)
        print(uri)
        break
PY
)"

EXPECT_PREFIX="${PUBLIC_SCHEME:-https}://gitlab.${BASE_DOMAIN:-daydaymoney.com}/users/auth/openid_connect/callback"
# 若环境未覆盖，RESOLVED 应等于默认公网 callback
if [[ -z "$RESOLVED" ]]; then
  echo "FAIL: empty redirect_uri from task-auth bootstrapClients" >&2
  exit 1
fi
if [[ "$RESOLVED" == *":8012"* ]]; then
  echo "FAIL: redirect_uri must not include :8012 (got $RESOLVED)" >&2
  exit 1
fi
if [[ "$RESOLVED" != http*://*/users/auth/openid_connect/callback ]]; then
  echo "FAIL: unexpected redirect_uri shape: $RESOLVED" >&2
  exit 1
fi

# docker-compose 不得再硬编码 host:port redirect
if grep -n 'redirect_uri:.*"http://#{ENV' "$ROOT/gitService/docker-compose.yml"; then
  echo "FAIL: docker-compose still hardcodes http://host:port redirect_uri" >&2
  exit 1
fi
if ! grep -q "GITLAB_OIDC_REDIRECT_URI" "$ROOT/gitService/docker-compose.yml"; then
  echo "FAIL: docker-compose missing GITLAB_OIDC_REDIRECT_URI" >&2
  exit 1
fi
# compose 注释不得含未转义的 ${scheme}（会破坏插值）
if grep -n '\${scheme}' "$ROOT/gitService/docker-compose.yml"; then
  echo "FAIL: docker-compose contains unescaped \${scheme}" >&2
  exit 1
fi

echo "OK redirect_uri=$RESOLVED"
