#!/usr/bin/env bash
# 回归：GitLab external_url 必须等于 conf publicUrl/allowedHost（公网 HTTPS、无 :8012）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CONF_READ="$ROOT/runAll/scripts/conf-read.py"
COMPOSE="$ROOT/gitService/docker-compose.yml"

EXPECTED="$(python3 "$CONF_READ" gitService --json | python3 -c '
import json,sys
d=json.load(sys.stdin)
u=(d.get("publicUrl") or d.get("allowedHost") or "").rstrip("/")
print(u)
')"

if [[ -z "$EXPECTED" ]]; then
  echo "FAIL: empty publicUrl/allowedHost from conf" >&2
  exit 1
fi
if [[ "$EXPECTED" == *":8012"* ]]; then
  echo "FAIL: conf public URL must not include :8012 (got $EXPECTED)" >&2
  exit 1
fi
if [[ "$EXPECTED" != http*://* ]]; then
  echo "FAIL: unexpected public URL shape: $EXPECTED" >&2
  exit 1
fi

if grep -nE "external_url 'http://\\\$\{GITLAB_EXTERNAL_HOST" "$COMPOSE"; then
  echo "FAIL: docker-compose still hardcodes http://host:port external_url" >&2
  exit 1
fi
if ! grep -q "GITLAB_EXTERNAL_URL" "$COMPOSE"; then
  echo "FAIL: docker-compose missing GITLAB_EXTERNAL_URL" >&2
  exit 1
fi
if ! grep -q "nginx\['listen_https'\] = false" "$COMPOSE"; then
  echo "FAIL: docker-compose must disable container TLS (edge terminates TLS)" >&2
  exit 1
fi
if ! grep -q "nginx\['listen_port'\]" "$COMPOSE"; then
  echo "FAIL: docker-compose must set nginx listen_port to mapped HTTP port" >&2
  exit 1
fi

# run.sh 须导出 GITLAB_EXTERNAL_URL
if ! grep -q 'GITLAB_EXTERNAL_URL=' "$ROOT/gitService/run.sh"; then
  echo "FAIL: run.sh must export GITLAB_EXTERNAL_URL" >&2
  exit 1
fi

echo "OK external_url_ssot=$EXPECTED"
