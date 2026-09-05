# 测试意图：超管微信分账单号与带缘由分账

## 对应功能意图

`billing_admin_profit_sharing_share.intent.md`

## 用例

1. staff GET 列表含 `wechat_transaction_id` / `wechat_profit_sharing_id`，响应 JSON 无 openid。
2. 非 staff POST share → 403。
3. staff 缺 reason 或短于 8 字 → 400，CreateOrder 调用次数 0。
4. staff pending 行 share → 200，本地 processing/finished，审计行 reason 与 actor。
5. 同一 Idempotency-Key 重放 → 200 `idempotent=true`，CreateOrder 仍 1 次。
6. 前端展示两列；pending/failed 有分账按钮；确认携带 Idempotency-Key。
7. 55 分订单台账佣金 3 分（5% ROUND_HALF_UP）→ CreateOrder amount=2（向下取整）；成功后 `commission_yuan_cents` 回写 2。
8. 订单已全额退款 → 409，CreateOrder 调用次数 0。
9. 微信 INVALID_REQUEST「分账金额超出最大分账比例…」→ HTTP 409，响应 `error` 为微信 message，不是 502。
