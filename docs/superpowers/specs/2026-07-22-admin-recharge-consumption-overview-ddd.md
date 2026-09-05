# DDD：充值消费情况总览

## 限界上下文

- **Billing（taskBill）**：账本聚合只读查询；无新聚合根写操作
- **Identity（Django accounts）**：用户展示名 enrichment

## 读模型

`UserRechargeConsumptionSummary{user_id, recharge_points, recharge_amount_yuan, unconsumed_points, consumed_points, commissionable_points}`

## 事件

纯查询 — 无领域事件（豁免登记）。
