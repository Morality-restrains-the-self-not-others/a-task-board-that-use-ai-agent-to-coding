# 订单电子发票测试意图

## 对应功能意图

`docs/intents/backend/billing_order_wechat_fapiao.intent.md`

## 测试点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 未支付订单申请开票 | 4xx，不写申请 |
| T2 | 已支付首次申请 | 201 pending；重复 pending 409；Idempotency-Key 回放 |
| T3 | 单位抬头无税号 | 400 |
| T4 | 管理员 approve | 调用开票缝，蓝票 issuing/issued，事件 ISSUE_ACCEPTED |
| T5 | 管理员 reject | 可再次申请 |
| T6 | 订单 GET invoices[] | 挂在 order_id 下 |
| T7 | 无蓝票退款 | 不调 reverse |
| T8 | 有蓝票全额未消耗退款 | reverse，不重开（剩余成交 0） |
| T9 | 有蓝票部分消耗后退款 | reverse + 按剩余成交重开 |
| T10 | 非员工访问 admin 审批 | 403 |
| T11 | 租户跨租户 order_id | 404 |
| T12 | 有蓝票退款 reverse 受理 | 红票/原蓝 `reverse_pending`；`reverse_confirm_hours=72`；截止=红票 created_at+72h；事件 REVERSE_PENDING |
| T13 | 冲红回调 REVERSED | 原蓝 `reversed`、红票 `issued`；不再要求确认 |
| T14 | 超过 72h 仍 pending | 读路径展示 `reverse_expired`，不写库 |
| T15 | 已支付但 `total_yuan_cents=0` 申请开票 | 400 `ErrInvoiceZeroAmount`，不写申请 |
| T16 | 零额订单存量 pending 审批 | 400，不落蓝票 |
