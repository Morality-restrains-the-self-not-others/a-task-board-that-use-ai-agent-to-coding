# NFR 澄清 — 管理员订单分账只读

- **日期**: 2026-08-22
- **价值流**: `docs/superpowers/plans/2026-08-22-admin-order-profit-sharing-value-stream.md`
- **默认等级**: L2；资金**写**路径不在本增量；读路径安全 L3（资金分配隔离）

## 路径分片键审视

| 路径 | 分片键 | 说明 |
|------|--------|------|
| GET `/api/system-admin/orders/{order_id}/` | 路径仅 `order_id` | **L0 管理端跨租户主键读**，与 `loadOrderByID` / ADR-0018 一致。人数极少。升级触发：管理端需按租户分片检索时强制 `tenant_id` query（本增量不改写路径） |
| GET `/api/tenant/{tenant_id}/billing/orders/{order_id}/` | `tenant_id` | **合适**：账单库租户隔离；本增量不改查询，只保证响应无分账键 |
| FE `/system-admin/order-records/` | 无 | L0 管理后台 |
| FE `/tenant/{tenant}/billing/orders/` | `tenant` | 合适，租户壳 |

`order_id` 是订单 Snowflake 主键，适合点查，不是租户分片键；管理端点查允许。

## 幂等性审视

| 路径 | 副作用 | 判定 |
|------|--------|------|
| 上述全部 GET | 无 | **L0** 只读；重复查询同一快照。展开按钮非写，不生成 Idempotency-Key。Anti-Replay-OK: 只读展开 |

无 HTTP 写、无 Kafka、无 Webhook、无 timer。分账**写入**仍是既有 `markOrderForProfitSharing`（本增量不改）。

## 类别定级

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L0/L2 | 管理端点查 L0；租户路径已带 tenant L2 |
| 数据一致性 | L0 | 只读已落库快照 |
| 安全 | L3 | 资金分配对租户不可见；staff 门禁；无 openid |
| 可用性 | L2 | 无分账返回空数组；404 未知单 |
| 性能 | L2 | `idx_profit_sharing_order`；单订单列表 |
| 可观测性 | L2 | order_id + share_count；禁 openid |

## 质量场景

1. 刺激：staff 展开有分账订单。响应：见接收方与金额。
2. 刺激：租户 GET 同单。响应：无 `profit_sharing` 键。
3. 刺激：member 调 admin 详情。响应：403。
4. 刺激：admin JSON。响应：无 openid 字段。

## 领域模型影响

只读值对象 `ProfitSharingSnapshot` 挂在管理员订单读模型上，不进入租户 `ResourceOrder` 公开 JSON。不改变分账聚合写边界。
