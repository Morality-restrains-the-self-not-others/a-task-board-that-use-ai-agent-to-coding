# NFR 澄清 — 任务帖配额来源与订单消耗归属

- **Date:** 2026-08-20
- **Value stream:** `docs/superpowers/plans/2026-08-20-task-post-quota-source-and-order-consumption-value-stream.md`

## 路径分片键审视

| 路径 | 分片 ID | 是否合适 | 动作 |
|------|---------|----------|------|
| GET `/api/tenant/{tenant_id}/billing/quotas/` | tenant_id | 是（租户账户） | 保持 |
| GET `/api/tenant/{tenant_id}/billing/orders/{order_id}/` | tenant_id + order_id | tenant_id 为库级隔离；order_id 为实体键 | loadOrder 双键 |
| SPA `/tenant/{tid}/billing/` | tenant | 是 | 保持 |
| SPA `/tenant/{tid}/billing/orders/` | tenant | 是 | 保持 |
| Internal consume-task-post-quota | tenant_id | 是 | 保持 |
| Outbox BILLING_TRANSACTION_CREATED | 载荷含 tenant/account | 消费侧既有 | payload 增 order_id，不以 tenant 作幂等键 |

可伸缩性：L2。配额/批次按 tenant 查询；grant 已有 `(tenant_id, resource_type)` 索引，补 `(tenant_id, order_id)`。

## 幂等性审视

| 路径 | 副作用 | 重复边界 | 幂等键 | 重放 | 等级 |
|------|--------|----------|--------|------|------|
| GET quotas / GET order | 可能触发 ensure lots 回填 | 每租户每订单行项至多 1 条 purchase grant | `(tenant_id, order_id, resource_type=task_post, source_kind=purchase)` 唯一 | 再 GET 空操作 | L3 |
| markOrderPaid | 写配额+批次 | 订单已 paid 直接成功 | 订单 status | 不重复发放 | L3（既有） |
| consume 创建/续存 | 扣批次+流水 | 每 task 一次创建/一次续存 | `task_post_quota:{task_id}` / `task_post_renewal:{task_id}` | 返回 idempotent | L3 |
| 退款清 remaining | 写批次 | 订单退款一次 | 既有退款状态机 | 再执行 remaining 已 0 | L3 |

禁止用 tenant_id 单独作为消耗幂等键。

资金/配额路径默认 L3：满足。

## 其他 NFR

| 类别 | 等级 | 说明 |
|------|------|------|
| 性能 | L2 | 配额拆分一次 SUM；订单 events 按 related_order_id 查询，建议 LIMIT 100 |
| 安全 | L3 | 租户隔离；日志禁止 PII，order_id/task_id 可记 |
| 一致性 | L3 | 消耗与流水同一事务 |
| 可观测性 | L2 | slog：lot_consumed / purchase_lot_issued / purchase_lots_backfilled |

## 领域模型影响

- 批次是配额扣减的一致性边界（与账户计数器同事务）。
- 订单是查询侧归属根，不是消耗聚合根。
