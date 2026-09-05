# NFR 澄清 — 按交易单号查询订单

- **日期**: 2026-08-21
- **价值流**: `docs/superpowers/plans/2026-08-21-system-admin-order-trade-no-query-value-stream.md`
- **默认等级**: L2；资金写路径不在本增量 → 幂等 L0

## 路径分片键审视

| 路径 | 分片键 | 说明 |
|------|--------|------|
| GET `/api/system-admin/orders/?order_number=` | 无租户路径键 | **L0**：沿用既有管理端跨租户列表；人数极少。查询为 UNIQUE/PK/`payment_ref` **精确等值**，不是全表 LIKE。升级触发：管理端 QPS 需按租户分片检索时改为强制 `tenant_id` query |
| GET `/api/internal/taskbill/admin/orders/?order_number=` | 无 | 同上 L0，internal secret |
| GET `/api/tenant/{tenant_id}/billing/orders/?order_number=` | `tenant_id` | **合适**：租户隔离 + 账单库归属；`order_number` 不是分片键 |
| FE `/system-admin/order-records/` | 无 | L0 管理后台；查询结果仍落在该页 |

`order_number` **不得**作为分片键。本增量允许它作为**管理端等值查找条件**（有 UNIQUE），与「禁止只拿 ORD 扫全表 / 公开路径参数」不冲突。

## 幂等性审视

| 路径 | 副作用 | 判定 |
|------|--------|------|
| 上述全部 GET | 无 | **L0** 只读；重复查询返回同一结果集。前端查询按钮不生成 Idempotency-Key（非写）。同步门闩仅防连点 |

无 HTTP 写、无 Kafka、无 Webhook、无 timer。

## 类别定级

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L0/L2 | 管理端 L0；租户列表 L2 已带 tenant |
| 数据一致性 | L0 | 只读 |
| 安全 | L2 | staff/租户门禁；精确匹配；长度上限 128 |
| 可用性 | L2 | 未命中 200 空列表 |
| 性能 | L2 | 等值条件走主键/UNIQUE；`payment_ref` 无独立索引，管理端精确查可接受。升级：`payment_ref` 查询变慢时补索引 |
| 可观测性 | L2 | 记录 query_len 与 total，不记录完整渠道号 |

## 质量场景

1. 刺激：超管粘贴四段号查询。响应：200 且仅该单。
2. 刺激：超管粘贴微信 `payment_ref`。响应：命中对应已支付订单。
3. 刺激：租户 A 用租户 B 的 order_number 调租户列表。响应：空列表。
4. 刺激：129 字符查询。响应：400。

## 领域模型影响

引入只读查询值对象 `TradeNoQuery`（展示号 | 主键 | payment_ref），不改变 `ResourceOrder` 聚合写边界。
