# 实施计划：系统管理租户详情

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-system-admin-tenant-detail-design.md`

## 事件契约任务

- [x] 无 MQ：纯查询；意图文档已记例外。

## Tasks

### T1 — GET tenant by id（taskTenantService）

- [x] Red: `admin_tenants_get_test.go`（403/404/200）
- [x] Green: `handleAdminTenantGet` 解析 `{id}`，`getCompanyByID` + 联系方式
- [x] 路由：`/tenants/{id}/` Go 1.24 通配

### T2 — GET tenant-quotas（taskBill）

- [x] Red: `admin_tenant_quotas_test.go`
- [x] Green: staff 闸 + 复用 quotas 装配；网关 URI；openapi

### T3 — GET tenant-workspaces（taskProjectService）

- [x] Red: `admin_tenant_workspaces_test.go`
- [x] Green: staff 闸 + `WHERE company_id=?` 无 mine；authz replace；网关 URI；openapi

### T4 — orders `tenant_id` 过滤（taskBill）

- [x] Red: 扩展 admin list orders 测例
- [x] Green: `doAdminListOrders` 调用 `listTenantOrders`

### T5 — FE 名称链接

- [x] Red: `SystemAdminTenantsPanel.test.js` 断言 `<a href>`
- [x] Green: 真实锚点

### T6 — FE 详情页

- [x] Red: `SystemAdminTenantDetail.test.js`
- [x] Green: 路由 + 三块并行 GET + 空态/错误 traceId + 返回链接
- [x] Sidebar：`/system-admin/tenants/` 前缀高亮用户管理

### T7 — 网关 / 所有权 / Swagger

- [x] `taskGateway/routes/routes.yaml` + `routes-apply`
- [x] `db/api_route_ownership.yaml`
- [x] 各服务 openapi.yaml

## 验证命令

```
cd taskTenantService && go test ./src -count=1 -run 'AdminTenant'
cd taskBill && go test ./src -count=1 -run 'AdminTenantQuota|AdminListOrders.*Tenant'
cd taskProjectService && go test ./src -count=1 -run 'AdminTenantWorkspace'
cd taskFE/app && npx vitest run src/components/SystemAdminTenantsPanel.test.js src/views/SystemAdminTenantDetail.test.js
```
