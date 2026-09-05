# DDD：系统管理租户详情（查询侧）

- **日期**: 2026-08-25
- **NFR**: `docs/superpowers/plans/2026-08-25-system-admin-tenant-detail-nfr-clarification.md`

## Bounded Contexts

| 上下文 | 服务 | 本增量角色 |
|--------|------|------------|
| Tenant | taskTenantService | 读 Company |
| Billing | taskBill | 读 Quota 视图 + ResourceOrder |
| Project | taskProjectService | 读 Workspace |

无新限界上下文。应用层为系统管理查询适配器，不把三域合成新聚合。

## Entities（既有）

- Company (`tenant_company`)
- Workspace (`project_workspace_entries`)
- ResourceOrder (`billing_resource_order`)

Quota 为只读视图（任务帖剩余 + GitLab 磁盘/流量），非独立生命周期实体。

## Domain Events

无。纯查询例外已在设计/意图文档写明。

## 端口

- `GetCompany(id)` — 已有 `getCompanyByID`
- `GetQuotaView(tenantID)` — 复用 quotas 装配
- `ListWorkspacesByCompany(companyID, limit, offset)` — 新查询（无 mine）
- `ListOrdersByTenant(tenantID, …)` — 已有 `listTenantOrders`

## 幂等键

不适用（无副作用）。
