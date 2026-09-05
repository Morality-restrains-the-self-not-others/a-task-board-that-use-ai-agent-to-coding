# 零额订单禁止开票

- **Date:** 2026-08-29
- **Status:** accepted（/goal 零交互采用）
- **Iteration:** zero-amount-invoice-block
- **Architecture:** 无架构变更（既有 InvoiceApplication 聚合上增加领域不变量）
- **ADR:** 沿用 [ADR-0034](../../adr/0034-order-wechat-fapiao.md)；不新开 ADR

## 成功标准

1. 订单 `total_yuan_cents <= 0` 时，租户订单详情**不显示**「申请开票」，改为说明「订单金额为 0 元，无法申请开票」。
2. `POST .../invoice-applications/` 对零额已支付微信订单返回 **400**，文案与前端一致，**不写** `billing_invoice_application`，**不发** `BILLING_INVOICE_APPLICATION_SUBMITTED`。
3. 管理员对零额订单的 pending 申请点击「已开具」同样 **400**，不落蓝票（存量申请兜底）。
4. `total_yuan_cents >= 1` 的已支付微信订单开票路径不变。

## 问题

微信电子发票与增值税发票均要求价税合计 > 0。当前 `canApply` 仅看 `status=paid`，零元订单（赠送、调账、异常落库）仍露出申请按钮；后端 `applyInvoiceApplication` 亦不校验金额。

## 方案（选定）

**领域不变量：可开票订单必须 `TotalYuanCents > 0`。**

- 判定 SSOT：`billing_resource_order.total_yuan_cents`（与订单 JSON `total_yuan` / `total_yuan_cents` 同源）。`<= 0`（含负值异常）一律禁止。
- 前端：`OrderInvoiceSection` 在已支付且可解析金额 ≤ 0 时隐藏按钮、展示只读说明（`Anti-Replay-OK: display-only`）。金额字段缺失时不误判为零（兼容不完整 mock），由后端兜底。
- 后端：`applyInvoiceApplication` 在确认 `paid` + 微信渠道之后、写申请之前拦截；`approveInvoiceApplication` 在 `loadOrder` 之后同样拦截。错误 `ErrInvoiceZeroAmount` → HTTP 400。
- 拒绝路径打结构化 warn 日志（`tenant_id` / `order_id` / `total_yuan_cents`），不含抬头 PII。

不采用：仅禁用按钮（可被绕过）；仅前端拦截；把零额与「未支付」共用 `ErrInvoiceOrderNotPaid`（文案误导）。

## 非目标

- 不改定价/下单逻辑（为何会出现 0 元订单）。
- 不自动作废已开具的零额蓝票（若有存量，另案处理）。
- 无新 API、无新表、无架构图更新。

## 意图 → 事件

拒绝开票不是被接受的业务意图，**不发 MQ 事件**（与既有未支付拒绝一致）。成功申请仍发 `BILLING_INVOICE_APPLICATION_SUBMITTED`。
