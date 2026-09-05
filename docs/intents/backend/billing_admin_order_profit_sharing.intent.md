# 功能意图：管理员订单详情返回分账快照

## 意图

平台员工在系统管理订单记录页展开一笔订单时，服务端返回该单已落库的分账记录：接收方用户 ID 与分账金额。租户订单详情不得包含分账字段。

## 角色

- 系统管理员 / 平台员工：跨租户读
- 租户成员：不得读取分账

## 行为

1. `GET /api/system-admin/orders/{order_id}/`：网关已验证 + `IsPlatformStaff`；按主键 `loadOrderByID`；200 含 `items`、`resource_consumption`、`profit_sharing`（数组，可空）。
2. `profit_sharing[]` 来自 `billing_profit_sharing` where `order_id`，字段：`receiver_user_id`、`amount_yuan`、`amount_yuan_cents`、`status`、`settle_after`、`settled_at`、`fail_reason`、`fail_trace_id`。不含 openid。
3. 未登录 401；非平台员工 403；订单不存在 404。
4. `GET /api/tenant/{tid}/billing/orders/{order_id}/` JSON **没有** `profit_sharing` 键，即使库中有分账行。
5. 写入仍由既有支付完成路径 `markOrderForProfitSharing` 负责；本意图不新写。

## 非目标

- 不改分账执行、比例、微信回调
- 不回填历史
- 不给租户推荐页增加订单级分账表

## 业务意图 → 事件对照

**无对应新事件（纯查询例外）**

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|----------|
| 管理员读取订单分账 | — | — | — | 只读；记录已在支付成功路径写入 |

## 变更记录

- 2026-08-22：新增管理员订单详情与租户隔离
- 2026-08-26：分账快照增加 `fail_trace_id`
