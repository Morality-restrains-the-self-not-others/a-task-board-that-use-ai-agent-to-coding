#!/usr/bin/env bash
# One-shot: export people tables from saas.sqlite3 → taskTenantService import APIs.
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

# Snowflake / bigint IDs must be strings in JSON — Go float64 loses precision above 2^53.
ID_KEYS = (
    "id", "user_id", "company_id", "workspace_id", "created_by_id",
    "group_id", "invitation_token",
)

def normalize(row, bool_keys=()):
    m = dict(row)
    for k in bool_keys:
        if k in m:
            m[k] = bool(m.get(k))
    for k in ID_KEYS:
        if k in m and m[k] is not None and m[k] != "":
            m[k] = str(m[k])
    for k in list(m):
        if m[k] is None:
            m[k] = ""
    return m

def post(path, payload):
    data = json.dumps(payload, ensure_ascii=False).encode()
    req = urllib.request.Request(base + path, data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    if secret:
        req.add_header("X-Internal-Secret", secret)
    with urllib.request.urlopen(req, timeout=60) as resp:
        print(path, resp.status, resp.read()[:200])

members = [
    normalize(r, bool_keys=("is_admin", "is_active"))
    for r in conn.execute("SELECT * FROM accounts_company_member")
]
post("/api/internal/tenant/members/import", {"items": members})

invs = [
    normalize(r, bool_keys=("is_admin", "is_accepted"))
    for r in conn.execute("SELECT * FROM accounts_invitation")
]
post("/api/internal/tenant/invitations/import", {"items": invs})

groups = [normalize(r) for r in conn.execute("SELECT * FROM accounts_company_group")]
gmembers = [normalize(r) for r in conn.execute("SELECT * FROM accounts_company_group_member")]
post("/api/internal/tenant/groups/import", {"groups": groups, "members": gmembers})
print("done", len(members), len(invs), len(groups), len(gmembers))
PY
