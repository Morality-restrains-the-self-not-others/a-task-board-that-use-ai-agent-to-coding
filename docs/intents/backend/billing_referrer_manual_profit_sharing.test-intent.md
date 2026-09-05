# 测试意图：推荐人冻结/可分账单与手动分账

## 覆盖

- 展示态：冻结 / 可分账 / 过期 / 已分账 / 失败（按 `paid_at` 15/30 天）用于聚合分桶
- `status=failed` 且支付未满 15 天 → display=`failed`、不可分账，**不得**为 `frozen`；渠道 `frozen_amount` 不含该佣金
- `status=failed` 且 15–30 天 → 仍可重试（shareable）
- GET 返回 `channels`，body 无 openid / `order_number` / `order_id`；同渠道聚合、他人渠道不可见
- POST share-channel 对该渠道 shareable 行 CreateOrder；无行 404；仅冻结 409
- POST `/{id}/share/` shareable 调用 CreateOrder 一次；冻结/过期 409；跨用户 404；已 finished 不二次打款
- 16 天 due pending **不会**被 `processPendingProfitSharings` 自动执行
- 26 天兜底仍执行（回归）
- 前端：可分账出现按钮；过期文案；双击只一次 POST

## 价值流

`docs/flows/value-stream-test-integration.wsd` T60；T59 改为「25 天兜底仍在，8/15 天 due 不再自动打款」

## 文件

- `taskBill/src/profit_sharing_referrer_display_test.go`
- `taskBill/src/handlers_referrer_profit_sharing_test.go`
- `taskBill/src/wechat_profit_sharing_scan_due_test.go`
- `taskFE/app/src/components/ReferralStatsPanel.test.js`
