# 价值流 — 资源订单号全局唯一

- **Date:** 2026-08-19
- **Design:** `docs/superpowers/specs/2026-08-19-billing-order-number-uniqueness-design.md`

## Related Value Streams

- `2026-08-14-billing-order-detail-page-value-stream.md` — 列表订单号链到详情；本增量不改路由，只改号码语义。
- `2026-08-11-billing-order-comments` — 正交。

本增量是对既有「创建订单 → 列表展示订单号 → 详情/支付」流的 **生成规则修正**，不是新用户旅程。

## 价值增量

| # | 增量 | 用户可见结果 | 验证 |
|---|------|--------------|------|
| 1 | 新单号码 = `ORD-日期-Snowflake` | 多租户同日下单号码不同；后缀=主键 | `TestGenerateOrderNumber*` / `TestCreateOrderNumberUniqueAcrossTenants` |
| 2 | 去掉 SELECT MAX | 高并发不再因日序号 1062 耗尽重试 | 既有并发测 + 新单不走 MAX |
| 3 | 列表长号可换行 | `ORD-20260818-003` 位上的链接仍可点进详情 | BillingOrders / SystemAdmin `break-all` |

触发用户：租户成员看账单；系统管理员跨租户对账。
