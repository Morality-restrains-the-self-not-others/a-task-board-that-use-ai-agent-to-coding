# 账单首页累计消耗/支付 测试意图

## 对应功能意图

`docs/intents/billing-dashboard-consumption-vs-payment.intent.md`

## 测试点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 用量消耗 + PayPal 充值 + 赠送 + 资源订单实付 | 累计消耗含用量与订单；累计支付含 PayPal 与订单、不含赠送 |
| T2 | API 同时返回 `*_points` 与 `*_cents` | 同值（分） |
| T3 | 前端读 `*_points` | 卡片按元展示（12345 → 123.45） |
| T4 | 前端仅有遗留 `*_cents` | 仍能展示，不回退成恒 0.00 |
| T5 | 卡片文案 | 累计消耗标明不含已退款/已取消订单；累计支付标明不含后台赠送 |
| T6 | month_start | 使用本地日历月初，不用 UTC `toISOString` 日期 |
| T7 | 用量 + paid/refunded/cancelled 三笔 resource_purchase | 消耗仅含用量与 paid 订单；refunded/cancelled 不计 |

## 实现位置

- `taskBill/src/handlers_billing_statistics_test.go`
- `taskFE/app/src/composables/useBillingDashboardStatistics.mapping.test.js`
- `taskFE/app/src/views/BillingDashboard.statisticsCards.test.js`
