# 实施计划 — 超管用户页租户 Tab

- **日期**: 2026-08-25

## 切片

- [x] **T1** taskTenantService：`GET .../tenants/` 鉴权/分页/搜索 Red→Green；`countCompanies` + `searchCompaniesPage`
- [x] **T2** 网关 `routes.yaml` 登记 URI + `routes-to-apisix.py`
- [x] **T3** FE：Tab + `?tab=tenants` 不拉用户列表（composable + 页面测）
- [x] **T4** FE：`SystemAdminTenantsPanel` 渲染/空态/错误 data-traceId/分页
- [x] **T5** `SystemAdminUsers.vue` Tab 抽组件，行数 ≤500；跑测

## 事件契约

无。publish/consumer 任务不适用（纯查询例外已写入意图文档）。
