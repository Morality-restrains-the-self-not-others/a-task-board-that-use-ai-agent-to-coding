# 前端订单发票测试意图

## 对应功能意图

`docs/intents/frontend/billing_order_wechat_fapiao.intent.md`

## 测试点

| ID | 场景 | 期望 |
|----|------|------|
| F1 | 已支付且无申请 | 显示申请开票按钮 |
| F2 | pending 申请 | 显示开票审批中，无按钮 |
| F3 | 已开蓝票 | 列表展示该票；退款后可见红字+新蓝 |
| F4 | 提交申请 | POST 带 Idempotency-Key |
| F5 | 管理端 Tab invoice | 渲染审批面板 |
| F6 | 报错 | data-traceId |
| F7 | reverse_pending / invoice_reverse_confirm | 显示 72 小时确认提醒与截止时间 |
| F8 | reverse_expired | 显示冲红确认超时文案 |
| F9 | 已支付且 total_yuan_cents=0 | 无申请按钮，显示无法开票说明 |
