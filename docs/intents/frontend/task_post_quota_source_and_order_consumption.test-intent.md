# 测试意图：账单与订单页配额来源及消耗展示

## 对应功能意图

`docs/intents/frontend/task_post_quota_source_and_order_consumption.intent.md`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| F1 | quotas 返回 gifted/purchased | 账单卡文案含赠送/购买数字 |
| F2 | 订单展开含 resource_consumption | 展开行含「资源消耗」与剩余 |
| F3 | 详情页 paid + consumption | 详情含消耗区块与 task_id |
| F4 | 超管退款批准/拒绝弹层 | 弹层含已发放/已消耗/剩余及逐笔 task_id |
| F5 | GitLab 磁盘消耗区块 | 展示「GitLab 磁盘」已发放数量，文案不含「任务帖：已发放 0」 |
| F6 | alwaysShow 且 consumption 为空 | 空态不含伪造的任务帖 0 |
