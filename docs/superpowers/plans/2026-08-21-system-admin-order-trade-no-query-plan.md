# 实施计划 — 按交易单号查询订单

- **日期**: 2026-08-21
- **设计**: `docs/superpowers/specs/2026-08-21-system-admin-order-trade-no-query-design.md`

## 事件契约

无 publish / 无消费者。查询例外已写入意图对照表。

## 切片

### Slice 1 — 后端查询（TDD）

- [ ] RED: `orders_list_trade_no_test.go`（展示号 / id / payment_ref / 空 / 超长 / staff 403 / 租户隔离 / 三段号不撞 id）
- [ ] GREEN: `listOrdersByTradeNo` + handler 读 `order_number`
- [ ] OpenAPI 增加 query 参数
- [ ] `gofmt` + `go test`

### Slice 2 — 前端查询（TDD）

- [ ] RED: 更新 `SystemAdminOrderListPanel.orderNumberPaste.test.js`
- [ ] GREEN: 交易单号输入走 GET 列表；空态；traceId
- [ ] 行数 ≤500；查询按钮 Anti-Replay-OK（只读）

### Slice 3 — 文档对照

- [ ] INDEX.md B-049f / F-093
- [ ] 修正 `billing_resource_order_number`「明确不做跨租户搜索」为：禁止公开扫描，允许 staff 等值查询
- [ ] value-stream-test-integration.wsd 测试点
