# 实施计划: PayPal 充值 Webhook 修复 + 架构加固

> 输入: `docs/specs/paypal-recharge-failure-fix/design.md`
> 价值流: `docs/superpowers/plans/2026-06-29-paypal-recharge-webhook-fix-value-stream.md`
> NFR: `docs/superpowers/plans/2026-06-29-paypal-recharge-webhook-fix-nfr-clarification.md`
> DDD: `billing_bridge/domain/events.py` (新增 3 个 PayPal 事件)

---

## 任务清单

### Phase 0: 运维操作 (P0)

- [ ] **T0.1** 更新 PayPal Developer Dashboard Sandbox Webhook URL
  - 文件: PayPal 后台 (无代码)
  - 操作: 将 webhook `8J8371221D9377308` 的 URL 从 `https://daydaymoney.com/...` 改为有效公网地址
  - 验证: PayPal Dashboard → Send Test Webhook → 检查 `saas-backend.log` 是否收到

- [ ] **T0.2** 同步配置文件
  - 文件: `conf/core/django/config.yaml`
  - 操作: 更新 `paypal.webhook_url_sandbox` 字段
  - 验证: `grep webhook_url_sandbox conf/core/django/config.yaml`

### Phase 1: Domain Layer (P1 — DDD 契约)

- [ ] **T1.1** 添加 PayPal 领域事件定义
  - 文件: `billing_bridge/domain/events.py` ✅ 已完成
  - 新增: `PayPalOrderCreated`, `PayPalOrderApproved`, `PayPalCaptureCompleted`
  - 验证: `from billing_bridge.domain.events import PayPalOrderCreated`

- [ ] **T1.2** 更新领域层导出
  - 文件: `billing_bridge/domain/__init__.py` ✅ 已完成
  - 验证: `python -c "from billing_bridge.domain import PayPalOrderCreated"`

- [ ] **T1.3** 注册 Kafka Topics
  - 文件: `core/kafka/config.py` ✅ 已完成
  - 新增: `PAYPAL_ORDER_CREATED`, `PAYPAL_ORDER_APPROVED`, `PAYPAL_CAPTURE_COMPLETED`
  - 验证: `python -c "from core.kafka.config import KAFKA_TOPICS; assert 'PAYPAL_ORDER_CREATED' in KAFKA_TOPICS"`

### Phase 2: Client Capture Fallback (P1 — 核心修复)

- [ ] **T2.1** `RechargePaypalStatusView.get()` 增加 capture 回退
  - 文件: `billing_bridge/recharge_views.py`
  - 改动: 当 status 为 pending 且可确认 order 归属当前用户时，调用 `process_checkout_order_approved_event(order_id)`
  - 约束:
    - 仅在 `pending.user_id == request.user.id` 时触发
    - 幂等安全（`_apply_paypal_credited_inner` 已有幂等保护）
    - 添加 Redis 锁防并发（`paypal_capture_lock:{order_id}` TTL 30s）
  - 验证: `pytest tests/test_paypal_client_side_capture_fallback.py -v`

- [ ] **T2.2** 添加频率限制
  - 文件: `billing_bridge/recharge_views.py`
  - 改动: 单用户每分钟最多触发 3 次 capture fallback
  - 实现: `cache.get(f'paypal_capture_rate:{user_id}')` 计数器
  - 验证: 单元测试确认超过限制返回 429

### Phase 3: 日志完善 (P1 — 可观测性)

- [ ] **T3.1** 创建订单成功日志
  - 文件: `billing_bridge/recharge_views.py` (RechargePaypalCreateView.post)
  - 新增: `logger.info("PayPal order created: order_id=%s amount=%s tenant=%s user=%s currency=%s", ...)`
  - 验证: `grep "PayPal order created" logs/saas-backend.log`

- [ ] **T3.2** Webhook 接收日志
  - 文件: `billing_bridge/paypal_webhook.py`
  - 新增:
    - `logger.info("CHECKOUT.ORDER.APPROVED received: order_id=%s", order_id)`
    - `logger.info("PAYMENT.CAPTURE.COMPLETED received: capture_id=%s", ...)`
  - 验证: 手动 webhook 测试后 `grep "CHECKOUT.ORDER.APPROVED received" logs/`

- [ ] **T3.3** Capture/Credit 结果日志
  - 文件: `billing_bridge/paypal_recharge.py`
  - 新增:
    - `logger.info("PayPal capture succeeded: order_id=%s status=%s", ...)`
    - `logger.info("PayPal credit succeeded: order_id=%s points=%s txn=%s", ...)`
  - 验证: capture 成功后检查日志

### Phase 4: 事件发布 (P1 — 领域事件)

- [ ] **T4.1** `RechargePaypalCreateView` 发布 PAYPAL_ORDER_CREATED
  - 文件: `billing_bridge/recharge_views.py`
  - 新增: `send_event('PAYPAL_ORDER_CREATED', {order_id, tenant_id, user_id, amount, currency})`
  - 验证: 单元测试 mock `send_event` 断言调用

- [ ] **T4.2** `paypal_webhook` 发布 PAYPAL_ORDER_APPROVED
  - 文件: `billing_bridge/paypal_webhook.py`
  - 新增: `send_event('PAYPAL_ORDER_APPROVED', {order_id, tenant_id, user_id, status})`
  - 验证: 单元测试

- [ ] **T4.3** `process_checkout_order_approved_event` 发布 PAYPAL_CAPTURE_COMPLETED
  - 文件: `billing_bridge/paypal_recharge.py`
  - 新增: `send_event('PAYPAL_CAPTURE_COMPLETED', {order_id, tenant_id, user_id, capture_status})`
  - 验证: 单元测试

### Phase 5: Outbox 重试 (P2)

- [ ] **T5.1** taskBill outbox 重试 goroutine
  - 文件: `taskBill/src/outbox_retry.go` (新建)
  - 实现: 后台 goroutine 每分钟扫描 `billing_outbox_message WHERE status='pending' AND created_at < now() - 1min`
  - 重试策略: 指数退避 1min→2min→4min→8min→16min→32min，最多 6 次
  - 验证: `go test taskBill/src/ -run TestOutboxRetry`

- [ ] **T5.2** outbox status 更新
  - 文件: `taskBill/src/credit.go`
  - 改动: `djangoEmitBillingEvent` 成功后更新 `billing_outbox_message.status = 'sent'`
  - 验证: 集成测试

### Phase 6: 前端 (P2)

- [ ] **T6.1** 轮询状态文案细化
  - 文件: `front_project/app/src/views/BillingRecharge.vue`
  - 改动:
    - "等待支付确认…" (0-5s)
    - "支付完成，等待入账…" (5-30s)
    - "服务端确认中…" (30-120s)
    - 120s 超时: 显示"服务端确认超时" + "手动刷新"按钮
  - 验证: Playwright E2E

- [ ] **T6.2** 手动刷新按钮
  - 文件: `front_project/app/src/views/BillingRecharge.vue`
  - 新增: 轮询超时后显示"手动刷新"按钮，重新触发状态查询
  - 验证: Playwright E2E

### Phase 7: 测试 (全覆盖)

- [ ] **T7.1** Client capture fallback 单元测试
  - 文件: `tests/test_paypal_client_side_capture_fallback.py` (新建)
  - 用例:
    1. Webhook 未到达 → 轮询触发 capture → 入账成功
    2. Webhook 已到达 → 轮询发现 completed → 不重复 capture
    3. 跨用户 order_id → 403
    4. 频率限制 → 超过 3次/min 返回 429
    5. 幂等 → 并发两次 capture → 仅入账一次

- [ ] **T7.2** 事件发布测试
  - 文件: `tests/test_billing_recharge_validation.py` (追加)
  - 用例:
    1. 创建订单 → mock 验证 `PAYPAL_ORDER_CREATED` 已发送
    2. Webhook → mock 验证 `PAYPAL_ORDER_APPROVED` 已发送
    3. Capture 成功 → mock 验证 `PAYPAL_CAPTURE_COMPLETED` 已发送

- [ ] **T7.3** 回归测试
  - 命令: `pytest tests/test_billing_recharge_validation.py -v`
  - 验证: 现有 12 个测试全部通过

### Phase 8: 集成验证

- [ ] **T8.1** 端到端手动测试
  - 使用测试账号完整的 PayPal 充值流程
  - 验证: capture → credit → 前端显示成功 → 交易记录可见

- [ ] **T8.2** Webhook 测试
  - PayPal Dashboard → Send Test Webhook
  - 验证: saas-backend.log 收到日志 + Kafka 事件已发布

---

## 依赖图

```
T0 (运维) ─────────────────────────────────────────────────────────────┐
                                                                        │
T1.1 → T1.2 → T1.3 (Domain Layer) ─── already done ───────────────────┤
                                                                        │
T2.1 → T2.2 (Capture Fallback) ─── 核心修复 ───────────────────────────┤
  │                                                                     │
  ├─→ T3.1, T3.2, T3.3 (日志) ─── 可并行 ─────────────────────────────┤
  ├─→ T4.1, T4.2, T4.3 (事件) ─── 可并行 ─────────────────────────────┤
  │                                                                     │
  └─→ T7.1, T7.2 (测试) ─── 依赖 T2+T3+T4 ────────────────────────────┤
                                                                        │
T5.1 → T5.2 (Outbox) ─── 独立 Go 改动 ─────────────────────────────────┤
                                                                        │
T6.1 → T6.2 (前端) ─── 独立前端改动 ───────────────────────────────────┤
                                                                        │
T7.3 → T8.1 → T8.2 (集成验证) ─── 最后 ───────────────────────────────┘
```

## 预估

| Phase | 工作项 | 估时 |
|-------|--------|------|
| P0 运维 | T0.1-T0.2 | 10 min |
| P1 Domain | T1.1-T1.3 | ✅ 已完成 |
| P1 Capture | T2.1-T2.2 | 30 min |
| P1 日志 | T3.1-T3.3 | 15 min |
| P1 事件 | T4.1-T4.3 | 20 min |
| P2 Outbox | T5.1-T5.2 | 45 min |
| P2 前端 | T6.1-T6.2 | 20 min |
| 测试 | T7.1-T7.3 | 30 min |
| 集成 | T8.1-T8.2 | 15 min |
| **总计** | | **~3h** |
