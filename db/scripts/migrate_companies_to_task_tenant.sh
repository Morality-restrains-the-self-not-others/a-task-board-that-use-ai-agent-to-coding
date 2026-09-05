#!/usr/bin/env bash
# One-shot: export accounts_company from saas.sqlite3 → taskTenantService import API.
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SAAS_DB="${REPO_ROOT}/db/saas/saas.sqlite3"
BASE="${TASK_TENANT_SERVICE_URL:-http://127.0.0.1:8020}"
SECRET="${INTERNAL_API_SECRET:-}"

if [[ ! -f "$SAAS_DB" ]]; then
  echo "missing $SAAS_DB" >&2
  exit 1
fi

python3 - <<'PY' "$SAAS_DB" "$BASE" "$SECRET"
import json, sqlite3, sys, urllib.request
saas, base, secret = sys.argv[1], sys.argv[2].rstrip("/"), sys.argv[3]
conn = sqlite3.connect(saas)
conn.row_factory = sqlite3.Row

def normalize(row):
    m = dict(row)
    for k in ("id", "creator_id"):
        if k in m and m[k] is not None and m[k] != "":
            m[k] = str(m[k])
    for k in list(m):
        if m[k] is None:
            m[k] = ""
    return m

items = [normalize(r) for r in conn.execute(
    "SELECT id, name, creator_id, created_at FROM accounts_company"
)]
payload = json.dumps({"items": items}, ensure_ascii=False).encode()
req = urllib.request.Request(base + "/api/internal/tenant/companies/import", data=payload, method="POST")
req.add_header("Content-Type", "application/json")
if secret:
    req.add_header("X-Internal-Secret", secret)
with urllib.request.urlopen(req, timeout=60) as resp:
    print(resp.status, resp.read()[:300])
print("done", len(items))
PY
