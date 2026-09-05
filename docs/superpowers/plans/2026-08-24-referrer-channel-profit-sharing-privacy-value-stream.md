# 价值流：推荐人按渠道看分账

增量：推荐人打开 `/profile/referral/` → GET 渠道聚合 → 看到渠道号/时段/订单金额/冻结/可分账 → 对可分账渠道点「分账」→ POST share-channel → 微信按单 CreateOrder。

测试点（T60 更新）：

- T60a GET 无 `order_number`/`order_id`
- T60b 同渠道多单聚合；不同渠道分行
- T60c 冻结渠道无按钮；可分账渠道有按钮
- T60d 双击只一次 POST share-channel
- T60e 他人渠道 404；空渠道 409
