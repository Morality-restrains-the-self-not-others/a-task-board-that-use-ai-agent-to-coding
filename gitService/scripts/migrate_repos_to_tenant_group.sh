#!/usr/bin/env bash
# 将租户下用户命名空间的 GitLab 仓库迁移至 tenant-{company_id} 组命名空间
# 用法:
#   bash gitService/scripts/migrate_repos_to_tenant_group.sh
#   bash gitService/scripts/migrate_repos_to_tenant_group.sh --tenant 850256677331562496
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONTAINER="${GITLAB_CONTAINER:-gitlab}"
PAT_FILE="${GITLAB_ADMIN_PAT_FILE:-${GITLAB_HOME:-$ROOT/gitService/gitlab_home}/.taskbill_admin_pat}"
if [[ ! -f "$PAT_FILE" && -f "$ROOT/gitService/gitlab_home/.taskbill_admin_pat" ]]; then
  PAT_FILE="$ROOT/gitService/gitlab_home/.taskbill_admin_pat"
fi
PROJECT_BASE="${TASK_PROJECT_SERVICE_BASE:-http://127.0.0.1:8016}"

TENANT_ID=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --tenant) TENANT_ID="${2:-}"; shift 2 ;;
    -h|--help)
      echo "用法: bash $0 [--tenant <id>]"
      echo ""
      echo "扫描租户 GitLab 仓库，报告位于用户命名空间（非 tenant-{id} 组）的仓库。"
      echo "对于不在组命名空间下的仓库，可通过 GitLab API transfer 移至组下。"
      echo ""
      echo "步骤："
      echo "  1. 确保 tenant-{company_id} 组已存在（运行 sync_tenant_gitlab_disk_quota.sh 自动创建）"
      echo "  2. 运行本脚本生成迁移报告"
      echo "  3. 对报告中的每个仓库，调用 GitLab API："
      echo "     POST /api/v4/projects/:id/transfer?namespace_id=<group_id>"
      exit 0
      ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

# 读取 PAT
if [[ -n "${GITLAB_ADMIN_PRIVATE_TOKEN:-}" ]]; then
  TOKEN="$GITLAB_ADMIN_PRIVATE_TOKEN"
elif [[ -f "$PAT_FILE" ]] && [[ -s "$PAT_FILE" ]]; then
  TOKEN="$(tr -d '\n' <"$PAT_FILE")"
else
  echo "[migrate-repos] 未找到 GitLab admin PAT，请先运行 sync_tenant_gitlab_disk_quota.sh --ensure-pat-only" >&2
  exit 1
fi

GITLAB_API_BASE="${GITLAB_API_BASE:-http://127.0.0.1:8012}"

query_params=""
if [[ -n "$TENANT_ID" ]]; then
  query_params="?tenant_id=${TENANT_ID}"
fi

echo "[migrate-repos] 从 taskProjectService 获取租户仓库列表..."
curl -sS "$PROJECT_BASE/api/internal/tenants/with-gitlab-local-repos/" \
  -H "X-Internal-Secret: ${TASK_PROJECT_SERVICE_INTERNAL_SECRET:-task-project-service-local-dev-secret}" 2>/dev/null \
  | python3 -c "
import json, sys
data = json.load(sys.stdin)
ids = data.get('tenant_ids', [])
print('\n'.join(ids))
" > /tmp/gitlab_tenants_with_repos.txt 2>/dev/null || {
  echo "[migrate-repos] 警告：无法从 taskProjectService 获取租户列表，请指定 --tenant" >&2
  exit 1
}

echo "[migrate-repos] 共 $(wc -l < /tmp/gitlab_tenants_with_repos.txt) 个租户有仓库"

while IFS= read -r tid; do
  tid="$(echo "$tid" | tr -d '[:space:]')"
  [[ -z "$tid" ]] && continue
  echo ""
  echo "=== 租户 $tid ==="

  # 查找 tenant-{id} 组的 ID
  GROUP_INFO=$(curl -sS -H "PRIVATE-TOKEN: $TOKEN" "$GITLAB_API_BASE/api/v4/groups?search=tenant-$tid&per_page=1" 2>/dev/null)
  GROUP_ID=$(echo "$GROUP_INFO" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d[0]['id'] if d else '')" 2>/dev/null || true)

  if [[ -z "$GROUP_ID" ]]; then
    echo "  组 tenant-$tid 不存在，请先运行 sync_tenant_gitlab_disk_quota.sh" >&2
    continue
  fi
  echo "  组 tenant-$tid 的 GitLab ID: $GROUP_ID"

  # 获取组下所有项目路径
  GROUP_PROJECTS=$(curl -sS -H "PRIVATE-TOKEN: $TOKEN" "$GITLAB_API_BASE/api/v4/groups/$GROUP_ID/projects?per_page=100" 2>/dev/null \
    | python3 -c "import json,sys; d=json.load(sys.stdin); print('\n'.join(p.get('path_with_namespace','') for p in d))" 2>/dev/null || true)

  # 列出所有项目，检查不在 tenant-{id} 组下的项目
  ALL_PROJECTS=$(curl -sS -H "PRIVATE-TOKEN: $TOKEN" "$GITLAB_API_BASE/api/v4/projects?per_page=100&membership=false&owned=true" 2>/dev/null \
    | python3 -c "
import json, sys
data = json.load(sys.stdin)
for p in data:
    ns = p.get('namespace', {})
    ns_path = ns.get('full_path', '')
    # 跳过已经属于 tenant-{id} 组的项目
    if ns_path.startswith('tenant-$tid'):
        continue
    # 跳过系统/内置项目
    if ns_path in ('root', 'admin'):
        continue
    print(f\"{p['id']}|{p['path_with_namespace']}|{ns_path}\")
" 2>/dev/null || true)

  if [[ -z "$ALL_PROJECTS" ]]; then
    echo "  所有仓库已在 group 命名空间下"
  else
    echo "  以下仓库不在 tenant-$tid 组命名空间下，需迁移："
    echo "$ALL_PROJECTS" | while IFS='|' read -r pid path ns; do
      echo "    - [$pid] $path (当前命名空间: $ns)"
    done
    echo "  迁移命令示例（逐个仓库执行）："
    echo "$ALL_PROJECTS" | while IFS='|' read -r pid path ns; do
      echo "    curl -X POST -H \"PRIVATE-TOKEN: \$TOKEN\" \"$GITLAB_API_BASE/api/v4/projects/$pid/transfer?namespace_id=$GROUP_ID\""
    done
  fi
done < /tmp/gitlab_tenants_with_repos.txt

echo ""
echo "[migrate-repos] 完成"
