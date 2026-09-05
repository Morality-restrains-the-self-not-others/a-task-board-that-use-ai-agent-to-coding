# 领域增量：冲红 72 小时购方确认窗口

- **Date:** 2026-08-23
- **Bounded context:** Billing / Invoice（taskBill）
- **Architecture:** 沿用 v100，无新服务

## 值对象

`ReverseConfirmWindow`

- `Hours` = 72（常量，与数电红字确认单一致）
- `Deadline` = 红票 `created_at` + 72h（UTC）
- `Expired` = now > Deadline（只读计算）

## 聚合变更

`Invoice` 状态机：

- reverse 受理：蓝 `issued|issuing` → `reverse_pending`；插入红票 `kind=red purpose=reverse status=reverse_pending`
- 回调 `FAPIAO.REVERSED`：蓝 → `reversed`；红 → `issued`
- 读路径超时：展示 `reverse_expired`，**不写库、无 ticker**

## 领域事件

| 事件 | 何时 | 载荷 |
|------|------|------|
| `BILLING_INVOICE_REVERSE_PENDING` | reverse API 受理 | order_id, invoice_id, red_invoice_id, confirm_deadline |
| `BILLING_INVOICE_REVERSED` | 微信回调确认 | order_id, wechat_apply_id |

## 读模型

订单 JSON `invoice_reverse_confirm`：required / hours / deadline / expired / message。
