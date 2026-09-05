#!/usr/bin/env bash
# 同步租户系统内建 GitLab 磁盘已用量到 taskBill，并下发 repository_size_limit 硬限额。
# 用法:
#   bash gitService/scripts/sync_tenant_gitlab_disk_quota.sh
#   bash gitService/scripts/sync_tenant_gitlab_disk_quota.sh --tenant 850256677331562496
#   bash gitService/scripts/sync_tenant_gitlab_disk_quota.sh --ensure-pat-only
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONTAINER="${GITLAB_CONTAINER:-gitlab}"
PAT_FILE="${GITLAB_ADMIN_PAT_FILE:-${GITLAB_HOME:-$ROOT/gitService/gitlab_home}/.taskbill_admin_pat}"
if [[ ! -f "$PAT_FILE" && -f "$ROOT/gitService/gitlab_home/.taskbill_admin_pat" ]]; then
  PAT_FILE="$ROOT/gitService/gitlab_home/.taskbill_admin_pat"
fi
TASKBILL_BASE="${TASKBILL_BASE:-http://127.0.0.1:8004}"
TASKBILL_SECRET="${TASKBILL_INTERNAL_SECRET:-}"
PROJECT_BASE="${TASK_PROJECT_SERVICE_BASE:-http://127.0.0.1:8016}"
GITLAB_API_BASE="${GITLAB_API_BASE:-http://127.0.0.1:8012}"

TENANT_ID=""
ENSURE_PAT_ONLY=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --tenant) TENANT_ID="${2:-}"; shift 2 ;;
    --ensure-pat-only) ENSURE_PAT_ONLY=true; shift ;;
    -h|--help)
      sed -n '2,8p' "$0"
      exit 0
      ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

ensure_admin_pat() {
  if [[ -n "${GITLAB_ADMIN_PRIVATE_TOKEN:-}" ]]; then
    echo "$GITLAB_ADMIN_PRIVATE_TOKEN" >"$PAT_FILE"
    chmod 600 "$PAT_FILE" || true
    echo "[disk-quota] using GITLAB_ADMIN_PRIVATE_TOKEN"
    return 0
  fi
  if [[ -f "$PAT_FILE" ]] && [[ -s "$PAT_FILE" ]]; then
    echo "[disk-quota] reusing PAT at $PAT_FILE"
    return 0
  fi
  if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
    echo "[disk-quota] GitLab container $CONTAINER not running; cannot create PAT" >&2
    return 1
  fi
  echo "[disk-quota] creating admin PAT via gitlab-rails..."
  mkdir -p "$(dirname "$PAT_FILE")"
  # Idempotent: revoke prior token by name then create.
  TOKEN="$(
    docker exec "$CONTAINER" gitlab-rails runner "
name = 'taskbill-disk-quota-sync'
u = User.find_by_username('root') || User.find_by(admin: true)
abort('no admin user') unless u
u.personal_access_tokens.where(name: name).find_each(&:revoke!)
t = PersonalAccessToken.new(user: u, name: name, scopes: %w[api read_api read_repository], expires_at: 1.year.from_now)
t.set_token(SecureRandom.hex(20))
t.save!
puts t.token
" 2>/dev/null | tail -n 1 | tr -d '\r'
  )"
  if [[ -z "$TOKEN" ]] || [[ "${#TOKEN}" -lt 16 ]]; then
    echo "[disk-quota] failed to create admin PAT" >&2
    return 1
  fi
  printf '%s\n' "$TOKEN" >"$PAT_FILE"
  chmod 600 "$PAT_FILE" || true
  echo "[disk-quota] wrote PAT to $PAT_FILE"
}

if ! ensure_admin_pat; then
  exit 1
fi

if $ENSURE_PAT_ONLY; then
  exit 0
fi

export GITLAB_ADMIN_PRIVATE_TOKEN
GITLAB_ADMIN_PRIVATE_TOKEN="$(tr -d '\n' <"$PAT_FILE")"
export GITLAB_API_BASE

BODY='{}'
if [[ -n "$TENANT_ID" ]]; then
  BODY="$(printf '{"tenant_id":"%s"}' "$TENANT_ID")"
fi

echo "[disk-quota] sync via taskBill $TASKBILL_BASE (project=$PROJECT_BASE gitlab=$GITLAB_API_BASE)"
HTTP_CODE="$(
  curl -sS -o /tmp/gitlab_disk_quota_sync.json -w '%{http_code}' \
    -X POST "$TASKBILL_BASE/api/internal/taskbill/sync-gitlab-disk-quotas/" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -H "X-TaskBill-Internal-Secret: $TASKBILL_SECRET" \
    -H "X-Internal-Secret: $TASKBILL_SECRET" \
    --data "$BODY"
)"
echo "[disk-quota] taskBill status=$HTTP_CODE"
if [[ "$HTTP_CODE" != "200" ]]; then
  head -c 500 /tmp/gitlab_disk_quota_sync.json; echo
  exit 1
fi
python3 - <<'PY'
import json
from pathlib import Path
p = Path("/tmp/gitlab_disk_quota_sync.json")
data = json.loads(p.read_text(encoding="utf-8"))
synced = data.get("synced") or []
enforce = data.get("enforce") or []
print(f"[disk-quota] synced={len(synced)} enforce={len(enforce)}")
for item in synced[:20]:
    err = item.get("error") or ""
    print(
        f"  tenant={item.get('tenant_id')} used={item.get('disk_used_bytes')} "
        f"quota_gb={item.get('disk_gb')} repos={len(item.get('repo_urls') or [])} err={err}"
    )
PY

# 新建 GitLab 仓库应创建在 tenant-{company_id} Group 下，以确保命名空间隔离和配额统一管理。
# GitLab API: POST /api/v4/projects?namespace_id=<tenant-group-id>&name=<repo-name>
# 迁移现有仓库：运行 gitService/scripts/migrate_repos_to_tenant_group.sh

# CE REST 不接受 repository_size_limit；经 gitlab-rails 下发 Group + Project 硬限额
if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  echo "[disk-quota] GitLab container not running; skip rails enforce" >&2
  exit 0
fi

echo "[disk-quota] applying repository_size_limit via gitlab-rails..."
python3 - <<'PY' | docker exec -i "$CONTAINER" gitlab-rails runner -
import json
from pathlib import Path
from urllib.parse import urlparse

data = json.loads(Path("/tmp/gitlab_disk_quota_sync.json").read_text(encoding="utf-8"))
items = data.get("enforce") or data.get("synced") or []

def project_path(repo_url: str) -> str:
    u = urlparse((repo_url or "").strip())
    path = u.path.strip("/")
    if path.endswith(".git"):
        path = path[:-4]
    return path

print("items = [")
for item in items:
    if item.get("error"):
        continue
    tid = str(item.get("tenant_id") or "").strip()
    limit = int(item.get("disk_limit_bytes") or 1)
    if limit <= 0:
        limit = 1
    repos = item.get("repo_urls") or []
    paths = []
    for r in repos:
        p = project_path(r)
        if p and "/" in p:
            paths.append(p)
    print("  {")
    print(f"    tenant: {json.dumps('tenant-' + tid)},")
    print(f"    limit: {limit},")
    print(f"    projects: {json.dumps(paths)},")
    print("  },")
print("]")
print(r'''
items.each do |item|
  path = item[:tenant]
  limit = item[:limit].to_i
  g = Group.find_by_full_path(path)
  unless g
    g = Group.new(name: path, path: path, visibility_level: Gitlab::VisibilityLevel::PRIVATE)
    g.save!
  end
  g.update!(repository_size_limit: limit)
  puts "group=#{path} limit=#{g.repository_size_limit}"
  Array(item[:projects]).each do |pp|
    p = Project.find_by_full_path(pp)
    unless p
      puts "project_missing=#{pp}"
      next
    end
    p.update!(repository_size_limit: limit)
    puts "project=#{pp} limit=#{p.repository_size_limit}"
  end
end
''')
PY

echo "[disk-quota] done"
