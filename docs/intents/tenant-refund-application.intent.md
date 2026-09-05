# 租户退款申请与管理员审批原路退回

## 意图

在租户账单页 `/tenant/{id}/billing/` 提供 **退款申请**：租户管理员提交后冻结全部可用积分并进入审批；系统管理员在 `/system-admin/order-records/?tab=refund` 审批（原独立页 `/system-admin/refund-applications/` 重定向至此）：

- **通过**：按支付台账 FIFO 对 PayPal/微信充值部分原路退款；核销冻结积分；写 `refund` 流水
- **驳回**：解冻积分恢复可用

- **Go taskBill**：冻结/台账/审批状态机/支付退款 API/出站事件
- **Django**：system-admin 超管闸门薄代理；租户路径经既有 `BillingProxyView`
- **Vue**：账单订单列表与**订单详情支付成功横幅**申请入口；确认弹层注明已消耗资源无法退回；system-admin「订单与退款」页退款 Tab（审批列表与「开启退款申请」开关）

## 验收

1. 租户管理员可对 **已支付的微信/PayPal 资源订单** 申请退款；非管理员 403；已有 pending 时 409
2. 订单直付（未入账户余额）也可申请：创建支付台账后按订单金额进入审批；**不**误要求账户余额 > 0
3. 超管可列表/通过/驳回；通过后原路退（WeChat/PayPal，可 mock）；驳回不误解冻无关余额
4. 赠送（admin_grant）/零金额订单不可申请；同订单不可重复 pending/approved
5. OpenAPI + `db/api_route_ownership.yaml` 已登记；架构制品齐全
6. 日志无卡号；关键路径带 request/trace id；业务错误返回 4xx（非 500「无可退金额」）
7. 超管可关闭退款申请：租户订单页「申请退款」按钮不可见；POST 返回 403 `REFUND_DISABLED`；切换「开启退款申请」开关须二次确认（取消则回滚且不保存）
8. `billing_payment_ledger` / `billing_refund_application` ID 列为 BIGINT，可容纳 Snowflake
9. 超管审批列表「关联订单」链接跳转 `/system-admin/order-records/?tenant_id={tid}&order_id=`（订单 Tab，非租户 `/billing/orders/`）后，管理端订单列表定位到该订单所在页并展开高亮
10. 订单详情已支付横幅可申请退款；文案标明已消耗资源无法退回；同单带 `Idempotency-Key` 重放返回已有申请
11. 超管批准/拒绝失败时，可读错误必须出现在弹窗内（含 `data-traceId`）；页面红条不得作为唯一出口（会被 `z-50` 遮罩挡住）；微信 SDK dump 与 `Wechatpay-Signature` 不得进入 JSON 与 UI

## 设计文档

- `docs/superpowers/specs/2026-07-22-tenant-refund-application-design.md`
- `docs/superpowers/specs/2026-07-22-tenant-refund-application-permission-analysis.md`
- `docs/superpowers/plans/2026-07-22-tenant-refund-application-value-stream.md`
- `docs/superpowers/plans/2026-07-22-tenant-refund-application-nfr-clarification.md`
- `docs/superpowers/plans/2026-07-22-tenant-refund-application-plan.md`
- `docs/superpowers/specs/2026-08-22-order-detail-refund-button-design.md`
- `docs/superpowers/specs/2026-08-22-order-detail-refund-button-permission-analysis.md`
- `docs/superpowers/plans/2026-08-22-order-detail-refund-button-nfr-clarification.md`

## 变更日期

2026-07-22；2026-08-19 修补订单直付退款（余额为 0 仍可按订单原路退）；2026-08-19 关联订单深链定位到列表所在页；2026-08-19 超管退款策略开关切换须二次确认；2026-08-19 超管退款审批并入订单查看页（旧路由重定向）；2026-08-19 超管退款审批列表「冻结金额」按元展示（API `frozen_points` 为分，UI 显示如 `0.55元`）；2026-08-19 超管「关联订单」改为管理端订单 Tab 深链（`?tenant_id=&order_id=`）；2026-08-22 订单详情支付成功横幅退款入口 + 已消耗不可退回说明 + Idempotency-Key 同单重放；2026-08-23 超管审批失败错误进弹窗并净化渠道 dump

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 退款申请已提交 | BillingRefundApplicationSubmitted | BILLING_REFUND_APPLICATION_SUBMITTED | taskBill outbox | task-events / 审计 | — |
| 退款申请已驳回 | BillingRefundApplicationRejected | BILLING_REFUND_APPLICATION_REJECTED | taskBill outbox | task-events / 审计 | — |
| 退款已完成 | BillingRefundCompleted | BILLING_REFUND_COMPLETED | taskBill outbox | task-events / 审计 | — |
| 退款流水 | BillingTransactionCreated | BILLING_TRANSACTION_CREATED | 既有 | 既有 | — |
