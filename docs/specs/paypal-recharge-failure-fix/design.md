# 设计文档：PayPal 充值失败 — 根因分析与修复方案

> 状态：待审批 | 日期：2026-06-29 | 类型：Bug 修复

---

## 1. 问题描述

用户 `bowek64234@divahd.com` 在租户 `850256677331562496` 充值页面选择 1000 元，
点击「跳转 PayPal 支付」并在 PayPal 沙箱批准后，跳回页面显示 **「充值失败」**。

---

## 2. 完整流程追踪

```
┌─ Stage 1 ─ 创建订单 ───────────────────────────────────────────┐
│  POST recharge_paypal_create/                                   │
│  → paypal_create_order(intent=CAPTURE, amount=1000, currency=HKD)│
│  → 返回 approval_url                                            │
│  → cache[paypal_recharge_pending:{order_id}] = pending info     │
│  事件: ❌ 无                                                    │
└─────────────────────────────────────────────────────────────────┘
     │ 用户跳转 PayPal 沙箱，登录 sb-vs9kf50794206@business.example.com
     │ 批准支付
     ▼
┌─ Stage 2 ─ 返回回调 ───────────────────────────────────────────┐
│  PayPal redirect → return_url?paypal_return=1&token=ORDER_ID    │
│  → handlePaypalReturnStatusPoll() 每 2s 轮询                    │
│  事件: ❌ 无                                                    │
└─────────────────────────────────────────────────────────────────┘
     │ 等待 Webhook...
     ▼
┌─ Stage 3 ─ Webhook (唯一 capture 入口) ─────────────────────────┐
│  PayPal POST → /api/billing/paypal/webhook/sandbox/             │
│  → CHECKOUT.ORDER.APPROVED                                      │
│  → paypal_capture_order(order_id)                               │
│  → _apply_paypal_credited_inner() → credit_recharge()           │
│  事件: ❌ 无 (在 webhook handler 中)                             │
└─────────────────────────────────────────────────────────────────┘
     │
     ▼
┌─ Stage 4 ─ 入账 (taskBill) ────────────────────────────────────┐
│  creditRecharge() → billing_transaction + billing_outbox_message │
│  → go djangoEmitBillingEvent() → Django emit_billing_event      │
│  → Kafka: BILLING_TRANSACTION_CREATED → billing-transaction-created│
│  → taskEvents consumer (port 18020) audit log                   │
│  事件: ✅ BILLING_TRANSACTION_CREATED                            │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. 根因分析

### 3.1 核心根因：PayPal Webhook URL 不可达 🔴 P0

PayPal 开发者后台注册的 Webhook URL：
```
https://daydaymoney.com/api/billing/paypal/webhook/sandbox/
```

**`daydaymoney.com` 域名 DNS 解析失败 — 该域名不存在。**

```
$ nslookup daydaymoney.com
*** Can't find daydaymoney.com: No answer
```

PayPal 服务器无法向此地址投递 Webhook → `CHECKOUT.ORDER.APPROVED` 从未到达 →
订单永不被 capture → 账户永不被 credit → 前端轮询 120s 超时 →「充值失败」。

验证：
- Django 直连端口 8001 的 webhook 端点正常（返回 400 invalid signature，符合预期）
- PayPal 沙箱 API 确认 webhook ID `8J8371221D9377308` 已注册，但 URL 指向幽灵域名
- 服务端日志零 webhook 记录、零 `recharge_paypal_status` 轮询记录

### 3.2 架构脆弱性：纯 Webhook 依赖 🔴 P1

当前 `intent: CAPTURE` 订单的 **capture 操作仅由 Webhook 触发**：

| 组件 | 职责 | 是否触发 Capture |
|------|------|:---:|
| `RechargePaypalCreateView` | 创建订单 | ❌ 只创建 |
| `RechargePaypalStatusView` | 查询状态 | ❌ 只读 |
| `paypal_webhook` | 接收回调 | ✅ 唯一入口 |
| `handlePaypalReturnStatusPoll` | 前端轮询 | ❌ 只读 |

没有 client-side capture 回退。Webhook 一断，全链路断。

### 3.3 领域事件缺口 🟡 P1

| 阶段 | 应有事件 | 实际 | 影响 |
|------|---------|------|------|
| PayPal 订单创建 | `PAYPAL_ORDER_CREATED` | ❌ 无 | 无法追踪/监控待支付订单 |
| Webhook 接收 | `PAYPAL_ORDER_APPROVED` | ❌ 无 | 无审批审计追踪 |
| Capture 成功 | `PAYPAL_CAPTURE_COMPLETED` | ❌ 无 | 无 capture 审计追踪 |
| 入账完成 | `BILLING_TRANSACTION_CREATED` | ✅ 有 | taskBill → outbox → Kafka |

**`BILLING_TRANSACTION_CREATED` 事件链路**：
```
taskBill creditRecharge()
  └→ INSERT billing_outbox_message (status='pending')
  └→ COMMIT
  └→ go djangoEmitBillingEvent()    ← goroutine, 异步 fire-and-forget
       └→ POST /api/internal/taskbill/emit-billing-event/
            └→ Django send_event('BILLING_TRANSACTION_CREATED', payload)
                 └→ Kafka topic: billing-transaction-created
                      └→ taskEvents consumer (port 18020) — 仅 audit log
```

**此链路的问题**：
1. **无重试机制**：`djangoEmitBillingEvent()` 是 fire-and-forget goroutine，失败后
   `billing_outbox_message.status` 永远保持 `pending`，无后台 Job 重试
2. **额外跳转**：taskBill → Django HTTP → Kafka，Django 不可达时 Kafka 事件丢失
3. **消费者是空操作**：taskEvents billing handler 只打 log（"taskBill already settled"），
   无实际副作用。所有业务逻辑已在 taskBill 完成

### 3.4 网关路由确认（无问题）

APISIX 路由配置正确（`routes.yaml:137-141`）：
```yaml
- id: billing-webhook
  priority: 830
  uri: /api/billing/*/webhook/*
  upstream: django
  auth_mode: none
```

Django 直连 webhook 端点返回 `400 invalid signature`（正确行为）。路由无问题，
纯粹是 PayPal 无法到达服务器。

### 3.5 日志缺口分析 🟡 P1

逐阶段对比「代码中的日志语句」vs「实际日志输出」：

| 阶段 | 代码中的日志 | 实际出现？ | 说明 |
|------|------------|:---:|------|
| ① 创建订单 | `http.access` (info) | ✅ 2条 | `POST 200` — 但**没有**记录 order_id / amount |
| ① 创建订单 | `logger.exception('PayPal 创建订单失败')` | ✅ 0条 | 仅在失败时输出 — 正常 |
| ① 创建订单 | **成功日志** | ❌ 缺 | 无 INFO 日志记录 order_id、amount、tenant |
| ② 返回轮询 | `http.access` (info) | ❌ 0条 | **前端从未轮询** `recharge_paypal_status` |
| ③ Webhook 到达 | `http.access` (info) | ❌ 0条 | PayPal 从未调用 webhook |
| ③ Webhook 到达 | `PayPal Webhook 签名校验未通过` (warning) | ✅ 1条 | 仅 07:10 手动测试时出现 — 端点工作正常 |
| ③ Webhook 到达 | **签名校验成功** (info) | ❌ 缺 | 无 INFO 日志记录 webhook 成功到达 |
| ④ Capture | `PayPal capture HTTP` (warning) | ❌ 0条 | 从未触发 capture |
| ④ Capture | **capture 成功** (info) | ❌ 缺 | 无 INFO 日志 |
| ⑤ 入账 | `PayPal 入账失败` (exception) | ❌ 0条 | 从未触发入账 |
| ⑤ 入账 | **入账成功** (info) | ❌ 缺 | 无 INFO 日志 |
| ⑤ taskBill | `emit billing event` (info) | ❌ 0条 | taskBill 日志无 credit/recharge 记录 |
| ⑤ taskBill | **outbox 状态更新** | ❌ 缺 | 无 outbox retry 日志 |

**关键发现**：

1. **前端从未轮询 `recharge_paypal_status`**（0 条 access log）。可能原因：
   - PayPal 返回 URL 的 `token` 参数未被正确解析
   - 用户手动离开页面而非等待跳回
   - `return_url` 的 `?paypal_return=1` + PayPal 追加 `?token=` 导致双 `?` 解析失败

2. **PayPal 创建订单返回 200 但无业务日志** — 不知道 order_id、审批 URL、
   金额等关键信息，排查只能靠 PayPal Dashboard

3. **Webhook 端点功能正常** — 07:10 的手动测试触发了日志 "签名校验未通过"，
   证明 Django 路由和 Webhook 处理链路完好，问题纯粹是 PayPal 投递不到

4. **全链路无成功 INFO 日志** — 所有日志都是 WARNING/ERROR 级别，
   正常路径无迹可寻，排查只能靠推断

### 3.6 附加上下文发现

- **06-28**：tenant `850256677331562496` 的 taskBill billing account 返回 404
  (`account not found`)，导致创建 Todo 时 500 错误（4次）。到 06-29 已修复。
- **06-29 01:44 / 02:22**：SMS 验证时 taskAuth 不可达（503），用户可能多次重试
  才完成验证。

---

## 4. 修复方案

### 4.1 立即修复：更正 Webhook URL 🔴 P0

在 [PayPal Developer Dashboard](https://developer.paypal.com/dashboard/applications/sandbox)
→ Webhooks → 编辑 webhook `8J8371221D9377308`，将 URL 改为 PayPal 可访问的公网地址：

**推荐方案**：配置有效域名 + HTTPS 证书
```
https://<真实域名>/api/billing/paypal/webhook/sandbox/
```

**临时方案**（测试用，PayPal 可能拒绝 HTTP）：
```
http://183.250.1.132:4000/api/billing/paypal/webhook/sandbox/
```

同步更新配置文件 `conf/core/django/config.yaml` 中的 `webhook_url_sandbox` 字段。

### 4.2 架构加固：客户端回退 Capture 🔴 P1

**文件**：`billing_bridge/recharge_views.py`

在 `RechargePaypalStatusView.get()` 中，当轮询发现订单仍为 pending 时，
服务端主动尝试 capture（Webhook 降级回退）：

```python
# 在 status 为 pending 且超过一定时间（如 10s）后
from billing_bridge.paypal_recharge import process_checkout_order_approved_event
ok, msg = process_checkout_order_approved_event(order_id)
```

**安全保证**：
- `_apply_paypal_credited_inner()` 有幂等保护（同 order_id 只入账一次）
- `paypal_capture_order()` 是幂等的（已 capture 的订单再次调用不会重复扣款）
- 仅在 Webhook 未到达时作为回退触发

### 4.3 领域事件补充 🟡 P1

**新增事件**：

| 事件 | 发布位置 | 触发时机 |
|------|---------|---------|
| `PAYPAL_ORDER_CREATED` | `RechargePaypalCreateView.post()` | PayPal 订单创建成功后 |
| `PAYPAL_ORDER_APPROVED` | `paypal_webhook()` | CHECKOUT.ORDER.APPROVED 收到 |
| `PAYPAL_CAPTURE_COMPLETED` | `paypal_recharge.py` | `paypal_capture_order()` 成功后 |

**事件 Payload 示例**：
```json
{
  "order_id": "xxx",
  "tenant_id": "850256677331562496",
  "user_id": "xxx",
  "amount_yuan": "1000",
  "currency": "HKD",
  "status": "APPROVED"
}
```

### 4.4 Outbox 重试机制 🟢 P2

**文件**：`taskBill/src/`（Go 服务）

增加后台 goroutine 定期扫描 `billing_outbox_message WHERE status='pending' AND created_at < now() - 1min`，重新调用 `djangoEmitBillingEvent()`。

### 4.5 日志完善 🟡 P1

在各阶段补充 INFO 级别业务日志，使正常路径可追踪：

| 位置 | 新增日志 | 内容 |
|------|---------|------|
| `recharge_views.py:222` | `logger.info` | `PayPal order created: order_id=%s amount=%s tenant=%s user=%s` |
| `paypal_webhook.py:68` | `logger.info` | `CHECKOUT.ORDER.APPROVED received: order_id=%s` |
| `paypal_webhook.py:76` | `logger.info` | `PAYMENT.CAPTURE.COMPLETED received: capture_id=%s` |
| `paypal_recharge.py:146` | `logger.info` | `PayPal capture succeeded: order_id=%s status=%s` |
| `paypal_recharge.py:128` | `logger.info` | `PayPal credit succeeded: order_id=%s points=%s txn=%s` |
| `paypal_recharge.py:136` | `logger.exception` | 保留（已有） |
| `client.py:63` | `logger.warning` | 保留（已有） |

### 4.6 前端改进 🟢 P2

**文件**：`front_project/app/src/views/BillingRecharge.vue`

- 轮询时展示更细粒度的状态文案
- 区分「等待支付确认」vs「支付完成，等待入账」vs「服务端确认超时」
- 增加「手动刷新状态」按钮

---

## 5. Domain Concept Inventory

| 概念 | 类型 | Context |
|------|------|---------|
| PayPal Order | Entity（外部） | Billing |
| Billing Transaction | Entity | Billing |
| Billing Account | Aggregate Root | Billing |
| Billing Outbox Message | Entity | Billing (taskBill) |
| Pending Recharge Cache | Value Object | Billing |
| `BILLING_TRANSACTION_CREATED` | Domain Event | Billing |
| `PAYPAL_ORDER_CREATED`（新） | Domain Event | Billing |
| `PAYPAL_ORDER_APPROVED`（新） | Domain Event | Billing |
| `PAYPAL_CAPTURE_COMPLETED`（新） | Domain Event | Billing |

---

## 6. Value Stream Impact

- **受影响流**：billing-recharge
- **修改文件**：
  - `billing_bridge/recharge_views.py` — client-side capture 回退 + 创建订单日志
  - `billing_bridge/paypal_recharge.py` — 新增事件发布 + capture/credit 日志
  - `billing_bridge/paypal_webhook.py` — 新增事件发布 + webhook 接收日志
  - `taskBill/src/credit.go` — outbox 重试
  - `front_project/.../BillingRecharge.vue` — 轮询文案改进
  - `conf/core/django/config.yaml` — webhook URL 更新
- **新增测试**：`test_paypal_client_side_capture_fallback.py`

---

## 7. 总结清单

- **Webhook URL**：立即更新 PayPal 后台 webhook URL 为有效公网地址（P0，运维操作）
- **Client Capture**：在 status 轮询中增加服务端主动 capture 回退（P1，代码 ~20 行）
- **领域事件**：补充 PAYPAL_ORDER_CREATED / APPROVED / CAPTURE_COMPLETED 三个事件（P1）
- **日志完善**：各阶段补充 INFO 级别业务日志，含 order_id、amount 等关键字段（P1）
- **Outbox 重试**：taskBill 增加 pending outbox 重试机制（P2）
- **前端文案**：优化轮询提示，增加手动刷新（P2）
