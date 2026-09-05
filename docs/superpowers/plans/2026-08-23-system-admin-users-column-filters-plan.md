# 实施计划：系统管理用户表列过滤器

## 切片

### Slice 1 — 后端本库列过滤

- [x] 红：`handlers_system_admin_list_filter_test.go`（email/role/login_method/date/is_active/id/phone + 非超管 403 + 与 q AND）
- [x] 绿：解析 query → SQL WHERE；LIKE 消毒；结构化日志只记键名
- [x] Validate：`go test` 该包相关测例

### Slice 2 — 后端跨服务列过滤

- [x] 红：tenant_company / referrer / has_profit_sharing 测例（mock 下游）
- [x] 绿：候选 ID 上限 5000 → batch hydrate → 内存过滤 → 再分页
- [x] Validate：Go 测例全绿

### Slice 3 — 前端表头过滤行

- [x] 红：`SystemAdminUsers.columnFilters.test.js`（第二行渲染、参数、重置）
- [x] 绿：`SystemAdminUsersFilters.vue` + composable 拼 query + 防抖
- [x] `SystemAdminUsers.vue` 保持 ≤500 行
- [x] Validate：vitest 相关文件 + 既有列测例回归
