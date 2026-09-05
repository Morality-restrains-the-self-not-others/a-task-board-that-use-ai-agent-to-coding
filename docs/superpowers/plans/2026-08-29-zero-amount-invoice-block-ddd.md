# 零额订单禁止开票 — 领域模型

NFR：`docs/superpowers/plans/2026-08-29-zero-amount-invoice-block-nfr-clarification.md`

## 限界上下文

**计费 Billing**（taskBill）。发票仍挂 `order_id`，不跨上下文。

## 不变量

`ResourceOrder.TotalYuanCents > 0` 是「可申请 / 可登记蓝票」的前置条件，与 `status=paid`、微信渠道并列。

## 实体（既有）

- **ResourceOrder**：`TotalYuanCents` 为金额 SSOT。
- **InvoiceApplication**：仅在订单可开票时创建。
- **Invoice**：approve 时 `amount_yuan_cents` 取订单总额；零额不得插入。

## 领域错误

`ErrInvoiceZeroAmount` = `订单金额为 0 元，无法申请开票`

## 领域事件

| 意图 | 事件 | 本增量 |
|------|------|--------|
| 提交开票申请（成功） | BILLING_INVOICE_APPLICATION_SUBMITTED | 不变 |
| 零额申请/审批被拒 | （无） | 书面例外：校验失败不是被接受的意图 |

## 端口

无新端口。继续走既有 `applyInvoiceApplication` / `approveInvoiceApplication`。
