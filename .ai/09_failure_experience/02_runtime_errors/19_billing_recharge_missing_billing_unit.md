# [运行时] 充值流水无 billing_unit（计费单元列仍为「-」）

## 现象

交易页消费行已能显示计费单元名后，**充值**行「计费单元」仍为 `-`。

## 根因

1. `creditRecharge` INSERT 未写 `billing_unit_id`（列允许 NULL）
2. `billing_unit` 表仅有 `post_creation` / `server_start`，无充值展示单元

## 修复

- 迁移 `004_seed_recharge_billing_unit.sql`：种子 `unit_type=recharge` 名「积分充值」，并回填历史 `transaction_type=recharge`
- `creditRecharge`：`ensureBillingUnit("recharge", …)` 后写入 `billing_unit_id`
- 单测：`credit_recharge_billing_unit_test.go`

## 验收

列表 API 中充值行：`billing_unit.name === "积分充值"`；前端列不再回落 `-`。
