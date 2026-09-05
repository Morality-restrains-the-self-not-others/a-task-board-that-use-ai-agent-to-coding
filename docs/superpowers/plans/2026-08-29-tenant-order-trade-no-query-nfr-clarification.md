# NFR 澄清：租户订单列表按交易单号 / 商户单号查询

- **日期**: 2026-08-29
- **价值流**: `docs/superpowers/plans/2026-08-29-tenant-order-trade-no-query-value-stream.md`

资金域列表查询：安全/隔离 L3；性能/可用性 L2；一致性 L2（读已提交行）。

## 路径分片键审视

| 路径 | 分片 ID | 判定 | 可伸缩性等级 | 结论 / 动作 |
|------|---------|------|-------------|-------------|
| GET `/api/tenant/{tenant_id}/billing/orders/?order_number=` | `tenant_id` | 合适 | L2 | 查询必须带路径 `tenant_id`；`listOrdersByTradeNo(tid, …)` 已 `AND tenant_id = ?`。`order_number` 不是分片键。 |
| `/tenant/:tenant/billing/orders/` | `tenant` | 合适 | L2 | 前端路由租户与 API 路径一致。 |

升级触发：单租户 `billing_resource_order` 热区 > 100 万行且该查询 P95 超 SLA 时，为 `wechat_transaction_id` / `out_trade_no` 补租户内等值索引（仍以 `tenant_id` 为分片键）。

## 幂等性审视

| 路径 | 副作用 | 等级 | 理由 |
|------|--------|------|------|
| GET 租户订单列表（含 `order_number`） | 无 | L0 | 纯查询；重复点击只重复读。前端门闩仅防连点，不发 Idempotency-Key。 |
| 查询按钮 | 无 | L0 | Anti-Replay-OK: 只读列表查询 |

禁止把本 GET 标成写路径。不引入 Kafka。
