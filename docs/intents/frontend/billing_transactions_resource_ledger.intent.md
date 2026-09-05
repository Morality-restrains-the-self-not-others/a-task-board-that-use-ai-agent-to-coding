# 功能意图：交易记录表展示资源流水账

## 背景与目标

`BillingTransactionsTable` 的「变动内容 / 剩余金额」读不出扣了什么、瞬时账目如何变化。改为「变动明细 + 瞬时账目」流水账列。

## 范围与边界

- 范围内：`BillingTransactionsTable`、Dashboard 最近交易、展示 helpers
- 范围外：新页面、改筛选语义、新轮询

## 约束与风险

- 金额一律元、两位小数；禁止裸分
- 无 `ledger_snapshot` 时由 `balance_before_points` / `balance_after_points` 回退
- 保持 `data-alias="BillingTransactionsTable"`
- 租户页 `billing.transactions.main` region 不变

## 验收标准

1. 表头含「变动明细」「瞬时账目」
2. 赠送行显示资源数量文案，不出现 `+0.00`
3. 现金行瞬时账目显示 before→after 元
4. 正负色：入账绿 / 消耗红 / 退款蓝

## 实施计划

1. `ledgerSnapshotDisplay` helper + 单测
2. 改表列并更新 grantDisplay / Dashboard 测试

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 例外理由 |
|---------|--------|---------|
| 浏览交易流水账 | — | 纯前端展示，无服务端状态变更 |
