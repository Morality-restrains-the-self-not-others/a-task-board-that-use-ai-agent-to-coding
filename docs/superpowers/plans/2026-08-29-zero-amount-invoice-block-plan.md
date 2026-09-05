# 零额订单禁止开票 — 实施计划

## 切片

- [x] **S1** 后端申请拦截（Red→Green）：`TestApplyInvoiceZeroAmountRejected`；`ErrInvoiceZeroAmount`；`applyInvoiceApplication`；`writeInvoiceApplyError` 400
- [x] **S2** 后端审批兜底：`TestApproveInvoiceZeroAmountRejected`；`approveInvoiceApplication` + handler 400
- [x] **S3** 前端：`OrderInvoiceSection` 零额隐藏按钮 + 说明；Vitest F9
- [x] **S4** 意图/OpenAPI/价值流图/既有设计补丁；1 分订单仍可申请回归

每切片后：`gofmt` / `py_compile`（若有）/ 相关单测绿。

## 意图 → 事件任务

- 成功申请：既有 `BILLING_INVOICE_APPLICATION_SUBMITTED`（不改契约）
- 零额拒绝：docs/intents 标明无事件
