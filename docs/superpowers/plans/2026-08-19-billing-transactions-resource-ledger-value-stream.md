# 价值流 — 交易记录资源流水账

Mapping the approved design into a value stream.

## 既有流

账单写路径（充值/消耗/赠送）已存在；本增量是**只读展示**，挂在租户账单查看流上。

## 增量（单切片即可交付）

| 增量 | 用户价值 | 范围 |
|------|----------|------|
| I1 流水账可读 | 看清扣了什么、瞬时余额/配额 | taskBill 列表 enrich + taskFE 表列 |

不拆第二增量（GitLab 区域剩余回放）——价值低于现金+任务帖，记 OPT。

## 步骤与测试

1. API 返回 change_display / ledger_snapshot — `taskBill/src/transaction_change_enrich_test.go`、`transaction_ledger_snapshot_test.go`
2. 表格渲染变动明细与瞬时账目 — `BillingTransactionsTable.grantDisplay.test.js`、`transactionChangeDisplay.test.js`
3. Dashboard 最近交易同步 — `BillingDashboard.grantDisplay.test.js`

## 字段

- `task-bill.billing_transaction.balance_before`
- `task-bill.billing_transaction.balance_after`
- `task-bill.billing_transaction.amount`
- `task-bill.billing_resource_grant.quantity`
- `task-bill.billing_account.task_post_quota`

## 测试点（WSD）

TP-LEDGER-1 赠送资源文案  
TP-LEDGER-2 现金瞬时 before→after 元  
TP-LEDGER-3 配额消耗非 0 元空壳  
TP-LEDGER-4 任务帖剩余回放顺序  

YAML 注册：`conf/value-stream.yaml` → `billing-transactions-resource-ledger`
