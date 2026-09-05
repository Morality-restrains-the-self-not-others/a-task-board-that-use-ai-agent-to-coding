# GitLab 磁盘价格套餐 — DDD 要点

## 限界上下文

- **计费（taskBill）**：PricingPackage、BillingAccount、BillingUnit 真源。

## 模型变更

| 概念 | 变更 |
|------|------|
| PricingPackage | + `GitlabDiskPointsPerGBPerMonth`（值对象语义：非负积分单价） |
| BillingAccount | + `LockedGitlabDiskPointsPerGBPerMonth`（开户/换套餐从套餐拷贝） |
| BillingUnit | + `gitlab_disk`（展示/流水标签，unit=`GB/月`） |

## 领域事件

本迭代**无新事件**（价目配置扩展，与续存字段先例一致）。后续月结扣费应投递 `BillingConsumptionRecorded`（或既有消费事件）并带 `unit_type=gitlab_disk`、`usage_amount=GB`。

## 仓储/SQL

所有读写 PricingPackage / BillingAccount 的 SELECT/INSERT/UPDATE 须包含新列。
