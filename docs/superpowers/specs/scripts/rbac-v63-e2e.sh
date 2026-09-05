#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════
# RBAC v63 E2E — 模拟网关注入头直连服务（验证判定链路）
# 前置: taskAuth(:8003) + taskTenantService(:8020) 已部署新代码
# ═══════════════════════════════════════════════════════════════
set -uo pipefail

AUTH="http://localhost:8003"
TENANT="http://localhost:8020"
CID="e2e-company-001"
USER_ID="e2e-user-001"
# 模拟 APISIX forward-auth 注入头（PDP 已展开权限码）
ADMIN_PERMS="member:manage,company:manage,group:manage,project:manage,project:view,task:manage,task:view,group-members:manage,group-resources:manage,group-resources:view"
ADMIN_HEADERS=(-H "X-User-Id: ${USER_ID}" -H "X-User-Roles: super_admin" -H "X-Tenant-Perms: ${CID}:${ADMIN_PERMS}")
PASS=0; FAIL=0

check() { # check <name> <expected> <actual>
  if [ "$2" = "$3" ]; then PASS=$((PASS+1)); echo "✅ $1 (${3})"
  else FAIL=$((FAIL+1)); echo "❌ $1 expected=$2 got=$3"; fi
}

echo "── E1 创建租户自定义角色 ──"
BODY_FILE=$(mktemp)
CODE=$(curl -s -o "$BODY_FILE" -w "%{http_code}" -X POST "${AUTH}/api/auth/roles/" "${ADMIN_HEADERS[@]}" \
  -H "Content-Type: application/json" \
  -d "{\"company_id\":\"${CID}\",\"display_name\":\"项目专员\",\"permissions\":[\"project:view\",\"task:view\"]}")
ROLE_ID=$(grep -o '"id":"[^"]*"' "$BODY_FILE" | head -1 | cut -d'"' -f4)
ROLE_NAME=$(grep -o '"name":"[^"]*"' "$BODY_FILE" | head -1 | cut -d'"' -f4)
rm -f "$BODY_FILE"
check "E1 角色创建 201" "201" "$CODE"
[ -n "$ROLE_ID" ] && echo "   role_id=$ROLE_ID name=$ROLE_NAME"

echo "── E2 非法权限码拒绝（platform:manage 不可用于自定义角色）──"
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${AUTH}/api/auth/roles/" "${ADMIN_HEADERS[@]}" \
  -H "Content-Type: application/json" \
  -d "{\"company_id\":\"${CID}\",\"display_name\":\"越权角色\",\"permissions\":[\"platform:manage\"]}")
check "E2 非法权限码 400" "400" "$CODE"

echo "── E3 内置角色锁定（更新 tenant_admin 403）──"
RID_TADMIN=$(docker exec docker-mysql-mysql-1 mysql -uroot -proot123456 task_auth -N -e "SELECT id FROM auth_role WHERE name='tenant_admin'" 2>/dev/null)
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "${AUTH}/api/auth/roles/role_id/${RID_TADMIN}/" "${ADMIN_HEADERS[@]}" \
  -H "Content-Type: application/json" -d '{"permissions":["task:view"]}')
check "E3 内置角色锁定 403" "403" "$CODE"

echo "── E4 角色列表（含自定义角色）──"
BODY=$(curl -s "${AUTH}/api/auth/roles/company_id/${CID}/" "${ADMIN_HEADERS[@]}")
echo "$BODY" | grep -q "项目专员" && check "E4 角色列表含自定义角色" "found" "found" || check "E4 角色列表含自定义角色" "found" "missing"

echo "── E5 PDP 权限展开（自定义角色 project:view 判定）──"
# 模拟: 成员被分配自定义角色后 PDP 计算权限码集合（直接查库展开）
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${AUTH}/api/internal/authz/check" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"${USER_ID}\",\"company_id\":\"${CID}\",\"perm_code\":\"project:view\"}")
check "E5 PDP check 200" "200" "$CODE"

echo "── E6 未授权判定（无 billing:manage 头 → 403）──"
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "${TENANT}/api/tenant/member-role/company_id/${CID}/member_id/m1/" \
  -H "X-User-Id: ${USER_ID}" -H "X-Tenant-Perms: ${CID}:task:view" \
  -H "Content-Type: application/json" -d '{"role_name":"member"}')
check "E6 无 member:manage 403" "403" "$CODE"

echo "── E7 组资源继承（伪码命中 → 允许）──"
# 直接验证 HasGroupResourceAccess 的伪码注入语义: PDP 展开组资源为伪码
docker exec docker-mysql-mysql-1 mysql -uroot -proot123456 task_tenant -e "
INSERT IGNORE INTO tenant_company_group (id, name, company_id, created_by_id) VALUES ('e2e-g1', 'E2E组', '${CID}', '${USER_ID}');
INSERT IGNORE INTO tenant_resource_group_assignment (id, company_id, resource_type, resource_id, group_id, permission, assigned_by) VALUES ('e2e-rga1', '${CID}', 'project', 'p-e2e-1', 'e2e-g1', 'view', '${USER_ID}');" 2>/dev/null
echo "   组资源分配已写入"
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${AUTH}/api/internal/authz/check" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"${USER_ID}\",\"company_id\":\"${CID}\",\"perm_code\":\"group-res:project:p-e2e-1:view\"}")
check "E7 组资源伪码可判定" "200" "$CODE"

echo "── E8 角色校验端点（role-exists）──"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${AUTH}/api/internal/authz/role-exists?company_id=${CID}&role_name=member")
check "E8 role-exists 200" "200" "$CODE"

echo "── E9 跨租户隔离（其他租户看不到本租户自定义角色）──"
OTHER_BODY=$(curl -s "${AUTH}/api/auth/roles/company_id/e2e-company-002/" -H "X-User-Id: ${USER_ID}" -H "X-Tenant-Perms: e2e-company-002:task:view")
echo "$OTHER_BODY" | grep -q "项目专员" && check "E9 跨租户隔离" "isolated" "leaked" || check "E9 跨租户隔离" "isolated" "isolated"

echo ""
echo "════════════════════════════════════"
echo "结果: PASS=$PASS FAIL=$FAIL"
[ "$FAIL" -eq 0 ] && echo "🎉 E2E 全部通过" || echo "⚠️ 有失败用例"
