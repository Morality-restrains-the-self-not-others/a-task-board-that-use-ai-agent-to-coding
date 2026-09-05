# Value Stream: PayPal 充值 Webhook 修复 + 架构加固

> Derived from design: `docs/specs/paypal-recharge-failure-fix/design.md`
> Date: 2026-06-29

## Value Summary

用户完成 PayPal 支付后系统可靠地完成 capture 和入账，Webhook 不可达时有客户端回退。

## Related Value Streams

- **recharge-sms-verification-status-fix**: sibling — 同一充值流程的 SMS 验证修复，本流聚焦 PayPal 支付环节
- **billing-account-event-driven-init**: dependency — 充值依赖 billing_account 存在，该流已修复 account 创建问题
- **increment4-billing-sse-go**: related — BILLING_TRANSACTION_CREATED 事件消费，本流的入账会触发该事件

## End-to-End Flow

```
用户点击「跳转 PayPal」→ 创建订单 → PayPal 沙箱批准 → 浏览器跳回
→ Webhook 到达(或回退 capture) → 入账 → 前端显示成功
```

## Value Increments

### Increment 1: Webhook URL 修复 (Thin Slice — P0)
**Value:** PayPal 能投递 Webhook 到服务器
**Scope:** 运维操作 — PayPal 后台更新 webhook URL + conf 文件同步
**Depends on:** nothing
**Test:** 手动验证 — PayPal Dashboard 发送测试 Webhook

### Increment 2: Client Capture 回退 (Core — P1)
**Value:** 即使 Webhook 延迟/失败，用户返回后 120s 内仍能完成 capture 和入账
**Scope:** `RechargePaypalStatusView.get()` 增加主动 capture 逻辑
**Depends on:** Increment 1
**Test:** `test_paypal_client_side_capture_fallback.py`

### Increment 3: 日志 + 事件补充 (Enhancement — P1)
**Value:** 全链路可追踪可监控，运维可排查
**Scope:** 3 个领域事件 + 5 条 INFO 日志
**Depends on:** Increment 2
**Test:** `tests/test_billing_recharge_validation.py` (已有) + 事件发布测试

### Increment 4: Outbox 重试 + 前端优化 (Enhancement — P2)
**Value:** Kafka 事件可靠性 + 用户可见状态提示
**Scope:** taskBill outbox retry goroutine + 前端文案
**Depends on:** Increment 3
**Test:** taskBill Go 测试 + Playwright E2E
