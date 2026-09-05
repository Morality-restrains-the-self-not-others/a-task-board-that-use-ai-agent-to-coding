# 功能意图：订单电子发票申请、审批开具与退款红冲重开

## 意图

已支付资源订单可申请电子发票；管理员审批后经微信支付开具并插入卡包。退款获批后全额红冲原蓝票，按剩余成交金额重开新蓝票。发票挂在订单下。

## 角色

- 租户管理员（`billing:manage`）：申请开票、查看本租户本单发票
- 租户成员（`billing:view`）：只读本单发票
- 平台员工（`IsPlatformStaff`）：审批申请、查看任意订单发票

## 行为

1. 订单 `status=paid`、`total_yuan_cents > 0`、无 pending 申请、无有效蓝票时，可 POST 申请（抬头：个人/单位，单位须税号）。`total_yuan_cents <= 0` 时 400「订单金额为 0 元，无法申请开票」，不写申请、不发事件。
2. 管理员 approve：调用微信开具；`fapiao_apply_id` = 该单微信支付 `transaction_id`。
3. 管理员 reject：申请结束，可再次申请。
4. 退款 approve 成功后：若有 `status=issued` 蓝票 → reverse（原因销货退回）→ 红票与原蓝票先标 `reverse_pending`，露出 72 小时确认截止；回调 `FAPIAO.REVERSED` 后再标完成。剩余成交金额 > 0 则新 `fapiao_id` 重开。
5. `GET` 订单 JSON 含 `invoices[]` 与 `invoice_reverse_confirm`（租户与管理员）。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 |
|---------|--------|--------|--------------|
| 提交开票申请 | BILLING_INVOICE_APPLICATION_SUBMITTED | applyInvoiceApplication | 审计/可观测 |
| 零额订单申请/审批被拒 | （无，校验失败） | applyInvoiceApplication / approveInvoiceApplication | 400，不写库 |
| 审批开具已受理 | BILLING_INVOICE_ISSUE_ACCEPTED | approveInvoiceApplication | 等待回调 |
| 蓝票开具完成 | BILLING_INVOICE_ISSUED | fapiao notify / 查询确认 | 订单发票列表 |
| 冲红已受理待购方确认 | BILLING_INVOICE_REVERSE_PENDING | 退款后 reverse API 受理 | 订单页 72h 提醒 |
| 蓝票全额红冲完成 | BILLING_INVOICE_REVERSED | FAPIAO.REVERSED 回调 | 原票失效 |
| 剩余金额重开 | BILLING_INVOICE_REISSUED | 重开受理/完成 | 新蓝票挂单 |

## 非目标

- 支付页微信官方开票入口作主路径
- 租户自助冲红
- 乐企行业数电 issue-general
