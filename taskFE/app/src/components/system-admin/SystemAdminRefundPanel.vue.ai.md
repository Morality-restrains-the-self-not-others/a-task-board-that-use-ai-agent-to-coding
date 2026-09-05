# SystemAdminRefundPanel.vue

超管退款审批列表。审批失败须把 `actionError` 传给批准/拒绝弹窗，禁止只写 `loadError`（会被 z-50 遮罩挡住）。渠道 dump 经 `humanizePaymentProviderError`。错误节点必须有 `data-traceId`。
