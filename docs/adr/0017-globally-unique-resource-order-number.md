# ADR-0017: 资源订单号由 Snowflake 派生、全局唯一

- **Status:** accepted
- **Date:** 2026-08-19
- **Author:** Trae AI
- **Deciders:** 工程团队

---

## Context

租户账单列表展示人读订单号（如 `ORD-20260818-003`）。该号码由 `generateOrderNumber` 按 **全表日序号** `ORD-YYYYMMDD-NNN` 分配：`SELECT MAX(...) WHERE order_number LIKE 'ORD-日期-%'`，再 `%03d` 递增，并依赖 `billing_resource_order.order_number` 全局 UNIQUE。

多租户放大后会出现：

1. **跨租户争抢同一日序号** — 所有租户共用一把全局计数器，并发下 SELECT MAX 读到同一值，已在生产触发 `Duplicate entry 'ORD-20260807-001'`（1062），靠 5 次重试收敛，租户一多即不够。
2. **3 位序号容量不足** — 全平台每日超过 999 单后格式膨胀、MAX+LIKE 更热。
3. **管理端歧义风险** — 系统管理员跨租户看订单。若改为「每租户独立 001、002」，不同租户会显示相同 `ORD-20260818-003`，客服/对账无法只凭号码定位一单。
4. **路由已用 Snowflake** — URL 是 `/tenant/{tid}/billing/orders/{id}/`，`id` 已是全局唯一 Snowflake；展示号不应再引入可碰撞的日序号。

主键 `id` 已是 Snowflake（`.ai/01_project_constraints/35_snowflake_id_generation.md`）。微信支付 `out_trade_no` 与订单号分离，不占用本格式。

## Decision

We will derive the human-readable `order_number` from the order's Snowflake primary key:

```
ORD-{YYYYMMDD}-{snowflakeID}
```

例：`ORD-20260818-877596007691485184`

约束：

1. **全局唯一** — 后缀与 `id` 相同，Snowflake 保证跨租户、跨日不碰撞；`UNIQUE(order_number)` 保留作防御。
2. **禁止 SELECT MAX 日序号** — 分配不再读表；生成是纯函数 `f(orderID, UTC date)`。
3. **存量号码不变** — 已落库的 `ORD-YYYYMMDD-NNN` 继续有效；仅新单使用新格式。
4. **UNIQUE 冲突仍重试** — `createOrder` / `insertOrderWithRetry` 在 1062 时换新 Snowflake 再生成号码（防御 MACHINE_ID 冲突）。
5. **长度** — `ORD-` + 8 位日期 + `-` + 最多 19 位十进制 ≈ 32 字符，落在 `VARCHAR(64)` 内。

## Alternatives Considered

### Alternative 1: 每租户日序号 + UNIQUE(tenant_id, order_number)

- **Pros:** 租户列表里仍是短号 `ORD-20260818-003`；争抢面从全表缩到单租户
- **Cons:** 管理端/客服/截图只看到同一字符串会指向不同租户；跨租户 UNIQUE 被拆掉后无法单号检索
- **Why rejected:** 用户明确问「租户多时是否冲突」；短号在多租户下语义就是会冲突

### Alternative 2: 加宽全局日序号（6 位）+ 继续 SELECT MAX

- **Pros:** 展示仍较短；改动面小
- **Cons:** 全表 MAX+LIKE 热点仍在；并发 1062 仍随租户线性恶化
- **Why rejected:** 不解决争抢根因

### Alternative 3: 仅展示 Snowflake、去掉 ORD- 前缀

- **Pros:** 最短全局唯一
- **Cons:** 丢失按日扫视；与现网 ORD- 习惯不一致
- **Why rejected:** 保留日期前缀对账更友好，且几乎不增加碰撞面

### Alternative 4: 号码中嵌入 tenant_id

- **Pros:** 人眼能看出租户；咨询只贴订单号即可解析租户
- **Cons:** 更长；截图含租户 ID（与 URL 相同）
- **Why rejected (original):** Snowflake 已全局唯一，URL 已有 tenant
- **Amendment:** 咨询场景需要「不另交租户 ID」。新单格式由 [ADR-0018](0018-order-id-tenant-shard-routing.md) 改为 `ORD-{日期}-{tenantId}-{snowflakeId}`。本 ADR 的全局唯一与禁止日序号仍然有效。

## Consequences

### Positive

- 任意多个租户同日下单，展示号不可能相同
- 分配路径无热点查询，并发不再依赖重试配额
- 客服可用订单号直接对应主键 `id`（后缀即 id）
- 管理端跨租户列表不会出现重复订单号

### Negative / Trade-offs

- 新单展示号从 ~16 字符变为 ~32 字符，列表需允许换行
- 新旧格式并存（存量 NNN vs 新 Snowflake 后缀）
- 不再有「该租户当日第 N 单」的短序号语义

### Mitigations

- 列表/详情使用 `font-mono break-all`
- 文档与意图写明两种格式均合法
- 需要租户内序号时另开显示字段，不占用全局 UNIQUE 的 `order_number`

## References

- 生产撞号回归：`taskBill/src/orders_duplicate_test.go`
- Snowflake 规范：`.ai/01_project_constraints/35_snowflake_id_generation.md`
- 设计：`docs/superpowers/specs/2026-08-19-billing-order-number-uniqueness-design.md`
- 新单展示号格式（嵌入 tenant_id、主键仍为末段 Snowflake）：[ADR-0018](0018-order-id-tenant-shard-routing.md)（accepted，修订本 ADR 格式条款）
