# 实施计划 — 管理员待分账订单列表

- **日期**: 2026-08-22
- **设计 / 权限 / 价值流 / NFR / DDD**: 同主题 `2026-08-22-admin-pending-profit-sharing-list-*`

## 事件任务（例外）

- [ ] 无新事件契约：纯查询；记录已在 `markOrderForProfitSharing`。意图文档已写例外。

## Tasks

- [x] **T1** Red：`handlers_admin_list_profit_sharing_test.go`
- [x] **T2** Green：queue + handler + mux + OpenAPI + `api_route_ownership.yaml`
- [x] **T3** 网关：`routes.yaml` + `routes-apply`
- [x] **T4** Red：tabs + panel vitest
- [x] **T5** Green：Tab + 面板
- [x] **T6** 价值流图测试点；gofmt/vet/yaml/vitest；登记精准重启

## 验收

`go test` 本包相关文件；vitest 上述 JS。
