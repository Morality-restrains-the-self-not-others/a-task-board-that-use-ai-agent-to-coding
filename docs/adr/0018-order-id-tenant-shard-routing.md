# ADR-0018: 资源订单展示号嵌入 tenant_id，主键仍为订单 Snowflake

- **Status:** accepted
- **Date:** 2026-08-19
- **Author:** Trae AI
- **Deciders:** 工程团队（/goal 自动采用）

---

## Context

[ADR-0017](0017-globally-unique-resource-order-number.md) 将展示号定为 `ORD-{yyyyMMdd}-{orderSnowflake}`，主键 `id` 与后缀相同，解决多租户日序号撞号。Snowflake **位布局不含租户**。租户面 URL 已有 `/tenant/{tid}/billing/orders/{id}/`。

新需求：用户/客服咨询时往往**只给订单号**，不愿或不便再交租户 ID。单库可用 `WHERE id=?` 反查租户；按 `tenant_id` 分片后，只拿订单 Snowflake **无法选片**。

ADR-0017 Alternative 4 曾因「URL 已有租户、号码变长」拒绝在展示号中嵌 `tenant_id`。咨询场景使该权衡失效。共片仍应使用 **列** `tenant_id`，而不是改全局 Snowflake 算法。

## Decision

We will keep `billing_resource_order.id` as the order Snowflake primary key (URL and FK unchanged) and encode tenant into the **display** `order_number` only:

```text
ORD-{UTC yyyyMMdd}-{tenantId}-{snowflakeId}
```

例：`ORD-20260818-877397588196749312-877596007691485184`

1. **Last segment = PK.** `snowflakeId` equals `id`. System identity does not become a composite string.
2. **Shard key remains `tenant_id` (column).** Do not hash-shard by `id`. Do not put tenant bits into the 64-bit Snowflake.
3. **Generate** `f(tenantID, orderID, UTC date)` with no `SELECT MAX`. `UNIQUE(order_number)` stays.
4. **Parse** by `-`: 4 segments → tenant + id; 3 segments → legacy ADR-0017 / daily `NNN` (no tenant gene).
5. **Lookup from a pasted number:** parse → `WHERE tenant_id=? AND id=?` → require row.tenant_id equals the gene. Gene is routing/hint only, not authorization.
6. **No anonymous public** “paste order number, get order” API. Admin/support tools may parse and jump to the existing tenant URL.
7. **Stock numbers unchanged.** WeChat `out_trade_no` unchanged. Length ≤ 52 ≤ `VARCHAR(64)`.

This **amends ADR-0017’s format clause** only. Global uniqueness and “no daily MAX sequence” still apply.

## Alternatives Considered

### Alternative 1: Keep ADR-0017 three-segment number (no tenant gene)

- **Pros:** Shorter; already shipping
- **Cons:** Support must also collect tenant ID, or PK-lookup which does not route after tenant sharding
- **Why rejected:** Explicit product need: consult with order number only

### Alternative 2: `ORD-{tenantId}-{date}-{snowflakeId}`

- **Pros:** Tenant first
- **Cons:** Harder to scan by day; weaker continuity with ADR-0017 date prefix
- **Why rejected:** Date-first matches existing ORD- habit

### Alternative 3: Instagram-style shard index in order Snowflake bits

- **Pros:** `id % N` equals tenant shard
- **Cons:** Not full tenant; freezes N; breaks unified 41+10+12 layout
- **Why rejected:** Wrong layer

### Alternative 4: Hash-shard by order `id`

- **Pros:** id-only PK routes to one shard
- **Cons:** Same tenant’s orders scatter
- **Why rejected:** Tenant is the scaling dimension

## Consequences

### Positive

- Pasting a new display number yields tenant + order id with no extra field
- After tenant sharding, new numbers are shard-routable without a locator table
- PK, URL, payment `out_trade_no`, and Snowflake generator stay stable
- Uniqueness still comes from the order Snowflake, not a daily counter

### Negative / Trade-offs

- Display number ~50 characters (lists need wrap; already `break-all`)
- Screenshots include tenant id (same as the page URL)
- Three formats coexist: `NNN`, ADR-0017 three-segment, four-segment
- Legacy three-segment numbers still cannot decode tenant

### Mitigations

- Parser branches on segment count
- Tampered middle segment fails the row tenant check
- Locator table only if legacy no-tenant lookup must survive sharding

## References

- 设计：`docs/superpowers/specs/2026-08-19-order-id-tenant-shard-gene-design.md`
- ADR-0017（唯一性仍有效；格式由本 ADR 修订）
- Snowflake：`.ai/01_project_constraints/35_snowflake_id_generation.md`
