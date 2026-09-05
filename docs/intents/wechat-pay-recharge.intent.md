# 微信支付 Native 充值

## 意图

为租户充值页 `/tenant/{id}/billing/recharge/` 接入 **微信支付 Native 扫码**通道，与现有 PayPal 并行：

- 用户选择微信支付 → 创建订单获得 `code_url` → 扫码支付 → 回调验签入账 → 前端轮询成功
- **Go taskBill** 负责预下单、回调验签解密、幂等入账（`wechat:{out_trade_no}`，`user_recharge_wechat`）
- **Django billing_bridge** 仅 SMS 门禁 + 薄代理（与 PayPal 同模式）
- 配置目录 `conf/billing/wechatPay/`（`pub_key.pem` + `pub_key_id`，公钥验签模式）
- SDK：`github.com/wechatpay-apiv3/wechatpay-go` + `WithWechatPayPublicKeyAuthCipher`
- `mode`: `live`（真实 Native 预下单）。缺凭证则禁用支付，不回退 mock；无 `mock-complete` HTTP 接口
- 回调：`POST /api/billing/wechat/notify/`

## 验收

1. `wechat_enabled=true` 时充值页展示微信支付选项与二维码
2. live：真实 notify 验签成功后入账；重复通知幂等
3. SMS 策略开启时，未验证用户无法 `recharge_wechat_create`
4. 密钥与完整回调体不出现在应用日志
5. Swagger + `db/api_route_ownership.yaml` 已登记；架构 v22 target 制品齐全
6. Native 下单 `amount.total` **单位为分**，等于订单 `total_yuan_cents`；禁止向上取整到整数元（0.55 元须下单 55 分，不得变成 100 分）

## 变更记录

- 2026-08-19：资源订单微信扫码按订单分金额原样下单。此前误假设「微信支付要求整数元」，把 0.55 元（任务帖）取整成 1.00 元。官方 APIv3 Native：`amount.total` 单位为分，1 元填 100。KYC 限额接口仍按整数元上取整，与微信金额解耦。

## 设计文档

- `docs/superpowers/specs/2026-07-14-wechat-pay-recharge-design.md`
- `docs/superpowers/specs/2026-07-14-wechat-pay-recharge-permission-analysis.md`
- `docs/superpowers/plans/2026-07-14-wechat-pay-recharge-value-stream.md`
- `docs/superpowers/plans/2026-07-14-wechat-pay-recharge-nfr-clarification.md`
- `docs/superpowers/plans/2026-07-14-wechat-pay-recharge-plan.md`

## 变更日期

2026-08-19


## 业务意图 → 事件对照

> 精修（2026-07-15）：事件名对齐仓库 MQ / domain events；同步写路径或非 MQ 副作用在例外理由标注「证据豁免」。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 微信充值支付成功入账 | BillingTransactionCreated | BILLING_TRANSACTION_CREATED | billing_bridge / 支付回调 | task-events billing_transaction_created | — |
