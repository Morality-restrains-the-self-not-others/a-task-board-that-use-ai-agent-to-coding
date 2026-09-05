# 实施计划 — 管理员订单分账展示

- **日期**: 2026-08-22
- **设计 / 权限 / 价值流 / NFR / DDD**: 同主题 `2026-08-22-admin-order-profit-sharing-*`

## 事件任务（例外）

- [x] 无新事件契约：纯查询；记录已在 `markOrderForProfitSharing`。意图文档已写例外。

## Tasks

- [x] **T1** Red：`handlers_admin_get_order_test.go` — staff 200 含 receiver/amount；空数组；401/403/404；租户 GET 无键；无 openid
- [x] **T2** Green：`listProfitSharingForOrder` + `handleSystemAdminGetOrder` + 注册 mux + OpenAPI
- [x] **T3** Red：`OrderExpandDetail.profitSharing.test.js` + 更新 `useAdminOrderRowExpand.test.js` URL
- [x] **T4** Green：组件分账区块；composable 改 admin GET；面板传入 `profitSharing`
- [x] **T5** 格式验证：gofmt/vet、py 不涉及、vue 测绿
- [x] **T6** 登记精准重启 `task-bill` `taskFE`

## 验收

`go test` 本包相关文件；vitest 上述 JS。
