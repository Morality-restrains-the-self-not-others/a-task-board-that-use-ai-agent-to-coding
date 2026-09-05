# DDD — 用户列表所属租户公司（只读投影）

- **日期**: 2026-08-23
- **NFR**: `docs/superpowers/plans/2026-08-23-system-admin-users-tenant-company-column-nfr-clarification.md`

## 限界上下文

- **Identity (taskAuth)**：拥有 `auth_user`；编排超管用户列表。
- **Tenant (taskTenantService)**：拥有 `tenant_company` / `tenant_company_member`。

跨上下文只走内部 HTTP 端口，不共享表。

## 模型

- **值对象** `TenantCompanyRef`：`id` + `name`。
- **应用服务** `fetchTenantCompaniesBatch`：端口 = 既有 `members/batch-get`；适配器在 taskAuth。
- **不新增聚合、不新增领域事件**。

## 业务意图 → 事件

纯查询。例外见 `docs/intents/backend/system_admin_users_list_tenant_companies.intent.md`。

## 端口

```
TenantMembershipLookup.BatchByUserIDs(ctx, userIDs) -> map[userID][]TenantCompanyRef
```

实现：HTTP 适配器，best-effort 空 map。
