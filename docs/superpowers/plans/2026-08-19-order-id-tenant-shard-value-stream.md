# 价值流 — 订单号嵌入租户基因

- **Date:** 2026-08-19
- **Design:** `docs/superpowers/specs/2026-08-19-order-id-tenant-shard-gene-design.md`

## Related Value Streams

- `2026-08-19-billing-order-number-uniqueness-value-stream.md` — 本增量**修改**其增量 1 的号码形态：`ORD-日期-id` → `ORD-日期-tenantId-id`。唯一性与禁止 MAX 不变。
- `2026-08-14-billing-order-detail-page-value-stream.md` — URL 仍用 Snowflake id，正交。

## 价值增量

| # | 增量 | 用户可见结果 | 验证 |
|---|------|--------------|------|
| 1 | 新单四段号 | 咨询只贴 ORD 号即可读出租户与主键 | `TestGenerateOrderNumber*` / Parse |
| 2 | 租户面复合加载 | 错租户 URL 404，不探测存在性 | IDOR + `loadOrder(tid,oid)` |
| 3 | 末段仍是 PK | 详情/支付 URL 不变 | 后缀 = `id` |

触发用户：租户看账单；客服只拿订单号；系统管理员跨租户对账。
