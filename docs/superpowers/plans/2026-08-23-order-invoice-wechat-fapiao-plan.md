# 订单电子发票实施计划

## 切片

- [x] **S0** DDL `dataMigrate/taskBill/061_order_invoice.sql` + 冷热清单注释（低频合规表，不分分区）
- [x] **S1** 领域类型与申请（Red→Green）：`invoice.go` `invoice_apply.go` `invoice_apply_test.go`
- [x] **S2** 审批开具缝：`invoice_approve.go` `wechat_fapiao.go` mock `issueWechatFapiaoFn`
- [x] **S3** HTTP + OpenAPI + APISIX + 订单 JSON `invoices[]`
- [x] **S4** 退款后 `ReconcileInvoicesAfterRefund`（T7–T9）
- [x] **S5** 回调 notify + 事件 publish
- [x] **S6** 前端 OrderInvoiceSection + 管理端 Tab
- [x] **S7** conf `fapiao`、意图文档已写、行数门禁
- [x] **S8** 冲红 72h 确认：`reverse_pending` + `invoice_reverse_confirm` + 订单页提醒（T12 F7）

每切片后：`gofmt` / `py_compile` / 相关单测绿。
