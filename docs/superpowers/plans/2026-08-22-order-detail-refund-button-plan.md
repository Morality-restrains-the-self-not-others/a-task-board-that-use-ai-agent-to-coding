# 实施计划：订单详情退款按钮

## 切片

- [ ] FE-1 共享 `BillingRefundConfirmModal`：消耗说明 + clickGuard 确认
- [ ] FE-2 `useBillingRefund.applyRefund` 发送 `Idempotency-Key`
- [ ] FE-3 `OrderDetail` 已支付横幅按钮/退款中/说明
- [ ] FE-4 `BillingOrders` 改用共享弹层（举一反三）
- [ ] BE-1 POST 读 `Idempotency-Key`；本单已有活跃申请则返回该申请
- [ ] DOC 意图 / 价值流 VS-R6..R9 / OpenAPI header

无新领域事件（沿用 `BILLING_REFUND_APPLICATION_SUBMITTED`）。
