# 测试意图：GitLab 磁盘目录价与起购 10 GB

## 对应功能意图

`docs/intents/backend/gitlab_disk_catalog_price_and_min_gb.intent.md`

## 测试目标

证明目录默认 4.00 元/GB/月，用户下单起购 10 GB，支付可发放 ≥10 GB。

## 测试分层

- 领域/无库：`gitlab_disk_qty_test.go`、`resource_pricing.go` JSON
- 基础设施：`resource_pricing_default_test.go` 迁移后单价；`gitlab_disk_min_purchase_test.go` 下单与支付

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 默认分值常量 | `DefaultGitlabDiskUnitPriceCents = 400`，`centsToYuanStr` 为 `4.00` |
| T2 | 迁移后读库 | `billing_unit.gitlab_disk.price = 400` |
| T3 | 缺行回退 | `getUnitPriceCents(gitlab_disk) = 400` |
| T4 | 数量 9 | `createOrder` 错误含「起购」与 `10` |
| T5 | 数量 10 | `createOrder` 成功 |
| T6 | 定价 JSON | `min_quantity=10`，无 `max_quantity` |
| T7 | 支付 10 GB | `markOrderPaid` 后该区域 `disk_gb >= 10`（额度入账；建组另见手动开通意图） |

## 数据与环境

- MySQL 测试库走 `dataMigrate/taskBill/`（含 076）
- 下单测例需 VIP1 与已启用区域 `tencent-sh-1`

## 通过标准

上述 Go 测例全绿。
