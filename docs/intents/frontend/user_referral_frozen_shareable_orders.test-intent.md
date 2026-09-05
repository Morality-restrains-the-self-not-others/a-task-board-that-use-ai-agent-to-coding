# 测试意图：推荐页冻结/可分账单

## 覆盖

- 表格列为渠道号/时段/订单金额/冻结/可分账，不含 `ORD-` 订单号
- 失败分账佣金不出现在冻结金额中（与管理端「失败」一致）
- 无可分账金额的渠道无分账按钮
- 可分账渠道有按钮，双击只 POST 一次 `share-channel`
- GET URL 为 referrer-orders，不请求 system-admin 分账接口

## 文件

`taskFE/app/src/components/ReferralProfitSharingOrdersPanel.test.js`
