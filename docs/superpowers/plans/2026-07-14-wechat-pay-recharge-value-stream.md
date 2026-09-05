# Value Stream: 微信支付 Native 充值

> Derived from design: `docs/superpowers/specs/2026-07-14-wechat-pay-recharge-design.md`
> Date: 2026-07-14

## Value Summary

租户成员在充值页选择微信支付，扫码完成后积分可靠入账，全程可轮询可见状态；无微信商户配置时仍可通过 mock 完成开发/E2E 验证。

## Related Value Streams

- **recharge-sms-verification** / **system-admin-phone-login-recharge-policy**：dependency — 创建订单前 SMS 门禁（策略开启时）
- **paypal-recharge-webhook-fix**：sibling — 并行在线充值通道，共享 SMS 与 credit-recharge 入账
- **increment4-billing-sse-go**：related — 入账触发 `BILLING_TRANSACTION_CREATED` 与前端 SSE 刷新

## End-to-End Flow

```
进入充值页 → SMS 验证（若策略要求）→ 选择「微信支付」→ 输入金额
→ 创建微信订单（code_url）→ 展示二维码 → 用户微信扫码支付
→ 微信 POST notify（验签解密）→ taskBill 幂等入账
→ 前端轮询 recharge_wechat_status → 展示充值成功
```

**mock 分支**（`mode=mock`）：

```
创建订单 → 展示 QR（或占位）→ internal mock-complete → 轮询 success
```

## Value Increments

### Increment 1: 配置 + mock 通路 (Thin Slice — P0)

**Value:** 无商户私钥时仍可端到端验证入账链路

**Scope:** `conf/billing/wechatPay/` 样例、`mode=mock`、internal mock-complete、Go 幂等 credit

**Depends on:** nothing

**Test:** Go `TestWechatMockCompleteCreditIdempotent`

### Increment 2: 创建订单 + 前端 QR (Core — P1)

**Value:** 用户可见微信二维码并发起支付

**Scope:** taskBill 预下单（live/mock）、Django `recharge_wechat_create`、BillingRecharge 微信 UI

**Depends on:** Increment 1

**Test:** Django SMS 门禁单测 + 前端组件测试

### Increment 3: 回调验签入账 (Core — P1)

**Value:** 真实/沙箱支付后自动入账

**Scope:** `POST /api/billing/wechat/notify/`、公钥验签、`wechat:{out_trade_no}` 幂等

**Depends on:** Increment 2

**Test:** Go 验签失败/成功/重复通知单测

### Increment 4: 状态轮询 + Playwright (Enhancement — P1)

**Value:** 用户无需刷新页面即可看到成功

**Scope:** `recharge_wechat_status`、前端 2s 轮询、Playwright mock E2E

**Depends on:** Increment 3

**Test:** `BillingRecharge.wechat-mock.playwright.test.js`

### Increment 5: Swagger / 架构 / value-stream.yaml (Enhancement — P2)

**Value:** 可治理、可审计、架构 SSOT 同步

**Scope:** openapi、api_route_ownership、v22 架构制品、`conf/value-stream.yaml` 登记

**Depends on:** Increment 4

**Test:** CI ownership check + 架构文件存在性

## Fields Impact

| 字段 / 配置 | 变更 |
|-------------|------|
| `billing_transaction.transaction_id` | 新增前缀 `wechat:` |
| `billing_transaction.points_source_type` | 新增 `user_recharge_wechat` |
| `recharge_phone_status` 响应 | +`wechat_enabled`, +`wechat_mode` |
| `conf/billing/wechatPay/*` | 新增配置目录 |
