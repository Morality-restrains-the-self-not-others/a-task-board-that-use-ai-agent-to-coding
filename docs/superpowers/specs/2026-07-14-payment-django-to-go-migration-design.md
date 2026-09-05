# 支付模块 Django → Go 迁移设计（2026-07-14）

## 目标
将用户充值支付（PayPal / 微信支付）业务逻辑从 Django `billing_bridge` 迁入 `taskBill`，Django 仅保留 SMS/管理员直充。

## 落点
| 能力 | Owner |
|------|--------|
| PayPal 创建/capture/webhook/验签/入账 | taskBill |
| WeChat Native prepay/notify/status/入账 | taskBill |
| SMS 发码/验证/phone_status | Django |
| 支付创建 SMS 门禁 | Django `BillingProxyView` |
| 公网 webhook / wechat notify | APISIX → taskBill（`auth_mode: none`） |
| Admin 直充 | Django → `credit-recharge` |
| PayPal 生命周期 Kafka | Go `emitPaypalLifecycleAsync` → Django `emit_billing_event(event_type=…)` |
| 入账 Kafka | 仍走 outbox → `BILLING_TRANSACTION_CREATED` |

## 已删除 Django 模块
- `paypal_service.py` / `paypal_recharge.py` / `paypal_webhook.py` / `wechat_recharge.py`
- 显式 `recharge_paypal_*` / `recharge_wechat_*` View 路由
- `PaymentWebhookProxyView` 与三条 webhook/notify 薄代理路由（2026-07-14 后续）

## 配置
- `conf/billing/paypal/config.yaml`
- `conf/billing/wechatPay/conf.yaml`
- 网关：`taskGateway/routes/routes.yaml`（`taskBill` upstream；`billing-webhook` / `billing-wechat-notify`）

## 验收
- Go: `TestWechat*` / `TestPaypal*`
- Django: `test_billing_recharge_validation`（SMS 门禁经 Proxy）
- Playwright: wechat-native + paypal-default
- Ownership：`db/api_route_ownership.yaml` webhook 前缀 `status: go`
