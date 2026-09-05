# 测试意图：租户退款申请与管理员审批原路退回

## 覆盖矩阵

| 验收点 | 单测 | API/集成 | Playwright |
|--------|------|----------|------------|
| 申请冻结 + pending | taskBill `refund_*_test.go` | — | 账单页申请 mock |
| 重复 pending 409 | 单测 | — | — |
| 非租户管理员 403 | 单测（mock resolveUserMember） | — | — |
| 驳回解冻 | 单测 | — | 超管驳回 mock |
| 通过 FIFO + mock 原路退 | 单测 | — | 超管通过 mock |
| balance 展示 frozen | — | — | Dashboard 展示冻结 |
| 超管门禁 | — | Django view 测或前端 403 | system-admin 页 |
| 关闭退款开关后按钮隐藏 + POST 403 | taskBill `TestApplyRefundApplicationDisabled`；vitest `refundEnabled`；vitest `SystemAdminRefundPanel.policy-confirm`（开关二次确认） | — | system-admin 订单页退款 Tab 开关（含确认弹窗） |
| 消费 FIFO 扣 ledger | `TestConsumePaymentLedgerFIFO` / `TestApproveRefundUsesRemainingAfterFIFOConsume` | — | — |
| 真实渠道退款（mock 回退） | `TestExecuteProviderRefundMockEnv`；PayPal capture 解析单测 | sandbox/live 手工 | — |
| 超管审批列表冻结金额按元显示 | vitest `SystemAdminRefundPanel.frozen-amount`；`formatYuanCents` | — | — |
| 详情页申请退款 + 已消耗说明 | vitest `OrderDetail.refund` / `BillingRefundConfirmModal` | — | — |
| 同单 Idempotency-Key 重放 | taskBill `TestApplyRefundApplicationIdempotentReplay` | — | — |
| 审批失败错误在弹窗内且无签名 dump | vitest `SystemAdminRefundPanel.action-error-modal`；`humanizePaymentProviderError`；taskBill `TestWechatPayClientErrorNOT_ENOUGH` | — | — |

## 关键用例 ID

- UT-REFUND-01 申请成功冻结
- UT-REFUND-02 重复申请冲突
- UT-REFUND-03 驳回解冻
- UT-REFUND-04 批准核销+ledger 扣减
- UT-REFUND-05 平台关闭退款时申请失败
- PW-REFUND-01 租户账单申请按钮与状态
- PW-REFUND-02 超管审批列表操作
- PW-REFUND-03 超管关闭退款后租户无申请按钮
- UT-REFUND-06 超管策略开关二次确认：确认才 PUT；取消回滚
- PW-REFUND-04 超管策略开关确认/取消交互
- UT-REFUND-07 列表 API order_id 对齐分页 + 前端展开高亮
- UT-REFUND-07b 超管「关联订单」href 为 `/system-admin/order-records/?tenant_id=&order_id=`（vitest `SystemAdminRefundPanel.order-deeplink`）
- UT-REFUND-08 超管退款审批 `frozen_points=55` 显示为 `0.55元`
- UT-REFUND-09 订单详情已支付横幅申请退款与已消耗说明
- UT-REFUND-10 同单带 Idempotency-Key 重放返回已有申请
- UT-REFUND-11 超管批准/拒绝失败：弹窗内可读错误 + `data-traceId`，不含 `Wechatpay-Signature`（vitest `SystemAdminRefundPanel.action-error-modal`；Go `wechat_pay_client_error_test`）
