# DDD：推荐人渠道分账视图

## 限界上下文

taskBill（分账记录与微信出站）为 owner。taskReferral 持有渠道名称；本增量推荐人看板只用 **channel_code**（渠道号），不跨库读名称。

## 聚合

- 写模型不变：`billing_profit_sharing` 一行一单。
- 读模型新增：`ReferrerChannelProfitSharingView`（渠道 + 时段 + 三类金额）。

## 领域事件

纯查询 GET 例外。POST share-channel 成功路径仍只更新行状态 + 出站微信 CreateOrder（与现网手动分账同一写路径，无新 MQ）。
