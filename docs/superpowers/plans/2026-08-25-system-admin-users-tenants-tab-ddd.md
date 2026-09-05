# DDD — 超管租户目录（只读查询）

- **日期**: 2026-08-25
- **NFR**: `docs/superpowers/plans/2026-08-25-system-admin-users-tenants-tab-nfr-clarification.md`

## 限界上下文

- **Tenant (taskTenantService)**：拥有 `tenant_company`；本增量的查询 owner。
- **Identity (taskAuth)**：创建者 email/phone；仅内部 HTTP。

跨上下文不共享表。

## 模型

- **实体** Company（已有 `companyRow`）：不新增聚合。
- **值对象** `AdminTenantListItem`：id, name, creator_id, email, phone, created_at, updated_at。
- **应用服务** `handleAdminTenants`：分页查询 + 联系方式投影。
- **不新增领域事件**。

## 业务意图 → 事件

纯查询。例外见 `docs/intents/backend/system_admin_tenants_list.intent.md`。

## 端口

```
AdminTenantDirectory.List(ctx, search, limit, offset) -> {items, total}
CreatorContactLookup.Batch(ctx, userIDs) -> map[userID]{email, phone}  // 既有 fail-open 适配器
```

实现：现有 `searchCompanies` / `countCompanies` + `fetchCreatorContactsFn`。taskTenantService 无独立 domain/ 包（与 tenant-options 一致，handler 编排）。
