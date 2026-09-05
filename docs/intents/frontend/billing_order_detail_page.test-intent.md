# 订单详情页测试意图

## 对应功能意图

`docs/intents/frontend/billing_order_detail_page.intent.md`

## 测试点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 创建订单成功 | `router.push` 到 `billing_order_detail`，params 含 tenant 与 orderId |
| T2 | 创建失败 | 不跳转，创建页展示错误且带 `data-traceId` |
| T3 | 详情页 GET 含 items | 展示资源中文名、数量、单价、小计 |
| T4 | 路由 `.../orders/create/` | name=`order_create`，不被 `:orderId` 吞掉 |
| T5 | 路由 `.../orders/{id}/` | name=`billing_order_detail` |
| T6 | 待支付详情页支付轮询 | 关闭弹窗/卸载后停止 GET 轮询（原 OrderCreate 契约迁入） |
| T7 | 支付弹窗 QR | 打开弹窗后 `renderQrToCanvas` 收到真实 canvas 与 `code_url` |
| T8 | 无模拟支付 | 弹窗不出现「模拟支付成功」与测试模式提示 |
| T9 | 已支付微信订单 | 横幅含「申请退款」与「已消耗的资源无法退回」 |
| T10 | 本单 pending 退款 | 显示「退款中」，无申请按钮 |
| T11 | 打开确认弹层 | 含消耗说明；确认 POST 带 Idempotency-Key |
| T12 | 摘要交易/商户单号 | 已支付订单展示 `wechat_transaction_id` / `out_trade_no`；缺值显示「—」 |

## 实现位置

- `taskFE/app/src/views/OrderCreate.contract.test.js`
- `taskFE/app/src/views/OrderDetail.contract.test.js`
- `taskFE/app/src/components/PayOrderModal.contract.test.js`
- `taskFE/app/src/router.billingOrderDetail.test.js`
- `taskFE/app/src/views/OrderDetail.refund.test.js`
- `taskFE/app/src/views/OrderDetail.tradeNos.test.js`
- `taskFE/app/src/components/BillingRefundConfirmModal.test.js`
