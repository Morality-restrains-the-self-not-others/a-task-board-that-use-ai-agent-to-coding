# 价值流：系统管理租户详情

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-system-admin-tenant-detail-design.md`

Mapping the approved design into a value stream.

## 受影响的既有流

- `system-admin-user-management` — 租户 Tab 目录是本流入口。

## 增量切片（按用户价值排序）

| # | 增量 | 用户可感知价值 | 范围 |
|---|------|----------------|------|
| 1 | 名称可点 | 从目录进入详情 URL | FE 链接 + SPA 路由 + header GET |
| 2 | 剩余资源 | 看到配额 | tenant-quotas API + 卡片 |
| 3 | 工作空间 | 看到空间列表 | tenant-workspaces API + 表 |
| 4 | 订单 | 看到订单并跳转全部 | orders?tenant_id= + 表 + href |

最小可交付：1+2+3+4 同一次交付（运营一次点击要看到三块；拆开发会残缺）。

## YAML 字段（三节）

- `task-tenant.tenant_company.id`
- `task-tenant.tenant_company.name`
- `task-bill.billing_account` 配额派生字段（只读）
- `task-project.project_workspace_entries.id`
- `task-bill.billing_resource_order.tenant_id`

## 测试文件

- `taskFE/app/src/components/SystemAdminTenantsPanel.test.js`
- `taskFE/app/src/views/SystemAdminTenantDetail.test.js`
- `taskTenantService/src/admin_tenants_get_test.go`
- `taskBill/src/admin_tenant_quotas_test.go`
- `taskProjectService/src/admin_tenant_workspaces_test.go`
