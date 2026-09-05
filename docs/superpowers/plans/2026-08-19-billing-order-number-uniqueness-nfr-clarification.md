# NFR 澄清 — 资源订单号全局唯一

- **日期**: 2026-08-19
- **价值流**: `docs/superpowers/plans/2026-08-19-billing-order-number-uniqueness-value-stream.md`
- **默认等级**: L2；订单创建为计费写路径 → 一致性/幂等按 L3 审视

## 路径分片键审视

| 路径 | 分片键 | 说明 |
|------|--------|------|
| `POST /api/tenant/{tenant_id}/billing/orders/` | tenant_id | 合适：租户隔离 + 账单库归属；order_number 不是分片键 |
| `GET /api/tenant/{tenant_id}/billing/orders/` | tenant_id | 合适；列表过滤必须带 tenant |
| `GET /api/tenant/{tenant_id}/billing/orders/{order_id}/` | tenant_id + order_id | tenant 分片；order_id Snowflake 实体键 |
| FE `/tenant/:tenant/billing/orders/` | tenant | 合适 |
| FE `/tenant/:tenant/billing/orders/:orderId/` | tenant + orderId | 合适；展示号不进路径 |
| 系统管理员订单列表（跨租户） | 无租户键 | L0：管理后台人数极少；升级触发：需按租户分片检索时再强制 tenant 过滤 |

`order_number` **不得**作为分片键或跨租户查找的唯一路径参数（避免「只拿 ORD-… 扫全表」）。

## 幂等性审视

| 路径 | 幂等键 | 说明 |
|------|--------|------|
| POST 创建订单 | 无请求级幂等键（现状） | 副作用：插入订单+行项。重复触发源：双击/超时重试。业务重复边界：**一次成功 POST = 一笔新订单**（允许用户连下两单）。禁止用 tenant_id 当幂等键。order_number 由新 Snowflake 派生，重复 POST 得到不同号码而非覆盖。 |
| insertOrderWithRetry / UNIQUE 1062 | 新 snowflake id | 重放语义：换 ID+号码再插，不更新已成功行 |
| 管理端赠送订单 | 既有 idempotency_key（admin_grant） | 与本次号码格式正交；相同 key 不得因换号算法而拆成两单 |
| GET 列表/详情 | — | 无副作用 L0 |

资金路径默认 ≥ L3：表级 `UNIQUE(order_number)` + `PRIMARY KEY(id)` 保证不会插入两行同一号码。

## 类别定级

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L2 | 去掉全表 MAX；按 tenant 列表；号码分配 O(1) |
| 数据一致性 | L3 | 计费订单插入；全局 UNIQUE |
| 安全 | L2 | 号码含 id，与 URL 一致；不新增跨租户按号查询 |
| 可用性 | L2 | 1062 仍重试；生成失败直接 4xx/5xx |
| 性能 | L2 | 不再 SELECT MAX LIKE |
| 可观测性 | L2 | 已有 `resource_order_created` / `resource_order_number_collision` |

## 质量场景

1. 刺激：两租户同时创建订单。响应：两个 `order_number` 不同，且各自后缀等于自己的 `id`。
2. 刺激：同租户连续 1000 单（超过旧 3 位上限）。响应：全部成功，无格式回退到 NNN 争抢。
3. 刺激：注入已占用 order_number。响应：重试后换新 Snowflake 号码成功。

## 领域模型影响

`OrderNumber` 从「日序号 VO」改为「由 OrderId 派生的展示 VO」；唯一性不变量从「当日全局递增」改为「等于 ORD-日期-OrderId」。
