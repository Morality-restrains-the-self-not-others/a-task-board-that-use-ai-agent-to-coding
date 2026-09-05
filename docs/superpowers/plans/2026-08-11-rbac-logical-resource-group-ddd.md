# DDD — 逻辑资源组 v72

- **Date:** 2026-08-11

## 聚合

- **ResourceGroupCatalog**（taskAuth）：`auth_resource_group` + `auth_resource_member`（系统种子）
- **RoleResourceGrant**（taskAuth）：`auth_role_resource_group`
- **SubjectRoleBinding**（taskTenant）：既有，不改

## 不变量

1. member 仅挂 `kind=ui_region`
2. `ui_region.parent_id` → `kind=page`
3. 页面组 ≠ 业务 `tenant_resource_group_assignment`

## 端口

- PDP：`fetchRoleResourceGroupCodes(roleNames, companyID) → []string`
- Enforce：`RequireRegion` / `HasRegion` / `HasPage`（shareLib）

## 事件

- 绑定变更 → 依赖既有 membership_rev / 角色更新路径触发缓存失效（P1 不强制新 Kafka topic；OPT 可补 RoleResourceGroupsChanged）
