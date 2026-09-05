# NFR 澄清 — 下单施工留言

- **日期:** 2026-08-19
- **价值流:** `docs/superpowers/plans/2026-08-19-billing-order-buyer-note-value-stream.md`
- **默认等级:** L2；资金下单写路径一致性按 L3 审视

## 路径分片键审视

| 路径 | 分片键 | 说明 |
|------|--------|------|
| `POST /api/tenant/{tenant_id}/billing/orders/` | tenant_id | 合适；留言随订单行写入同租户分片 |
| `GET /api/tenant/{tenant_id}/billing/orders/{order_id}/` | tenant_id + order_id | 合适；Snowflake 实体键 |
| FE `/tenant/:tenant/billing/order-create/` | tenant | 合适 |
| FE `/tenant/:tenant/billing/orders/:orderId/` | tenant + orderId | 合适 |
| 超管展开同一 GET | tenant_id（URL） | 合适；禁止无租户扫留言 |

无新路径缺分片键。`buyer_note` 不是分片键。

## 幂等性审视

| 路径 | 副作用 | 重复边界 / 键 | 重放语义 |
|------|--------|----------------|----------|
| POST 创建订单（含可选留言） | 插入订单+行项+buyer_note | 一次成功 POST = 一笔新单；禁止 tenant_id 作幂等键 | 双击产生两单，可带相同留言 |
| GET 详情 | 无 | L0 | — |
| 无新 Kafka 消费 | — | — | — |

资金路径 ≥ L3：UNIQUE(id)/UNIQUE(order_number) 保证不覆盖已成功行。留言无独立幂等键，与订单插入同事务。

## 类别定级

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L2 | 列级 TEXT，无新表爆炸 |
| 数据一致性 | L3 | 与订单同事务 |
| 安全 | L3 | 长度上限、他租户 404、前端转义、日志禁正文 |
| 可用性 | L2 | 非法留言 400，不创建订单 |
| 性能 | L2 | 无额外查询 |
| 可观测性 | L2 | `buyer_note_len` + 既有 `resource_order_created` |

## 质量场景

1. 刺激：磁盘订单 + 500 字留言。响应：201/200 且详情可见全文。
2. 刺激：仅任务帖 + 非空留言。响应：400，无订单行。
3. 刺激：留言 2001 字。响应：400。
4. 刺激：他租户 GET 本单。响应：404，body 无留言。

## 领域模型影响

`ResourceOrder` 增加值对象式字段 `BuyerNote`（可空串）；履约分类是领域服务谓词，不是新聚合。
