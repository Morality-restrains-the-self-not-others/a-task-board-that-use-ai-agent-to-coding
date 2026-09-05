# 测试意图：交易记录表资源流水账

## 测试目标

表格与 helper 把 API 字段渲染成可读流水账。

## 测试分层

- `taskFE/app/src/utils/transactionChangeDisplay.test.js`
- `taskFE/app/src/components/BillingTransactionsTable.grantDisplay.test.js`
- `taskFE/app/src/views/BillingDashboard.grantDisplay.test.js`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| F1 | change_display 任务帖 +10 | 表文案含该串，不含 `+0.00` |
| F2 | ledger_snapshot.display_lines | 「瞬时账目」列含各行 |
| F3 | 仅有 balance_before/after points | helper 格式化为 `余额 1.00 → 0.70 元` |
| F4 | Dashboard 最近交易 | 含瞬时账目，赠送不显示 +0.00 |

## 通过标准

Vitest 全绿。
