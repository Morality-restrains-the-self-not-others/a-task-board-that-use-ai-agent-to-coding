# 交易记录资源流水账可读性 — 设计文档

- **Date:** 2026-08-19
- **Status:** accepted（goal-mode 自动采用）
- **Author:** cursor
- **入口:** `/tenant/{tenantId}/billing/transactions/`（`BillingTransactionsTable`）

## 问题

租户「交易记录」表已有时间、类型、分类、变动内容、剩余金额，但仍无法当流水账读：

1. **看不出扣了什么**：配额入账/消耗 `amount_points=0` 时，变动列常为 `—` 或 `+0.00 元`；后台赠送未展开 `billing_resource_grant`。
2. **看不出瞬时账目**：`balance_after_points` 以分为单位裸展示，没有「变动前 → 变动后」；资源配额瞬时剩余完全缺失。
3. `_wip_aside/transaction_change_enrich.go` 已有展示 enrich，但未接入 `handleTransactionsList`。

现网页需登录，本次以源码与既有失败经验（`.ai/09_failure_experience/02_runtime_errors/17_billing_transactions_unit_id_not_nested.md`）为准。

## 目标（完成标准）

1. 每一行能读出**发生额 + 标的**（元 / 任务帖 / GitLab 磁盘 / 流量等）。
2. 每一行能读出**该瞬时点账目**：现金余额 before→after（元）；若该行影响任务帖配额，同时给出配额剩余。
3. 既有筛选、分页、RBAC 不变；仅扩展既有 GET 响应可选字段（向后兼容）。
4. 单元测试覆盖 API enrich 与前端展示；无新 Python 接口。

## 选定方案

**扩展既有列表 DTO + 前端流水账列**（不新开 endpoint、不改写路径、不新增表）。

| 方案 | 结论 |
|------|------|
| A. 仅改前端格式化 | 拒：配额行没有 `change_display` / `resource_changes`，无法展示「扣了什么」 |
| B. 新资源流水表 + 写路径快照 | 拒：本增量过重；现金已有 `balance_before/after` |
| **C. 查询侧 enrich + 列表改列（采用）** | 复用 WIP enrich；现金快照用已存字段；任务帖剩余按账户当前配额 + 未过滤历史回放 |

### API（taskBill，存量 GET）

`GET /api/tenant/{tenant_id}/billing/transactions/`  
`GET /api/tenant/{tenant_id}/billing/transactions/list_filtered/`

每行新增可选字段（缺省时前端可从 `amount_points` / `balance_*` 回退）：

```json
{
  "points_source_type_display": "后台赠送",
  "change_display": "任务帖 +10 帖",
  "resource_changes": [
    {
      "resource_type": "task_post",
      "quantity": 10,
      "remaining_after": 10,
      "label": "任务帖",
      "display": "任务帖 +10 帖"
    }
  ],
  "ledger_snapshot": {
    "balance_before_points": 100,
    "balance_after_points": 70,
    "balance_before_yuan": "1.00",
    "balance_after_yuan": "0.70",
    "display": "余额 1.00 → 0.70 元",
    "display_lines": ["余额 1.00 → 0.70 元", "任务帖剩余 9"]
  }
}
```

处理顺序：嵌套 `billing_unit` →（filtered）项目/成员名 enrich → `attachTransactionDisplayFields` → `attachLedgerSnapshots`。

任务帖瞬时剩余：读 `billing_account.task_post_quota`，再取「页面最旧行时刻起、该账户**未过滤**流水」按时间倒序回放 `resourceDelta`（赠送 +qty / 配额消耗 −usage）。筛选页不得只用当前页回放。

GitLab 磁盘/流量区域剩余回放本增量不做（记 OPT）。

### UI

| 列 | 展示 |
|----|------|
| 变动明细 | `transactionChangeDisplay`（带正负色） |
| 瞬时账目 | `ledgerSnapshotDisplay`：现金 before→after；资源剩余分行 |

Dashboard「最近交易」同步加「瞬时账目」，避免两处语义分裂。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 租户查看交易流水账 | — | — | — | 纯查询，无系统事实变更 |

## 领域概念（供 DDD）

- Bounded context: Billing
- Read model: `LedgerSnapshot`（现金 + 可选资源剩余）
- 已有聚合：`BillingTransaction`、`BillingResourceGrant`、`BillingAccount`
- 无新领域事件

## 价值流影响

影响租户账单「看清扣费/配额」路径；不新增充值/扣费写路径。测试：`transaction_change_enrich_test.go`、`transactionChangeDisplay.test.js`、`BillingTransactionsTable.grantDisplay.test.js`。

## 🏛️ 架构变更影响

**不更新 `docs/architecture/`。** 无新服务、无新数据所有权、无新跨服务流；仅扩展既有 taskBill 列表 JSON 与 taskFE 表格列。当前基线仍为 v85。

## 🕸️ Code Review Graph 分析

- `code-review-graph update --brief`：26 files, 0 nodes/edges 变化；图规模 Nodes 108 / Files 17（以 JS/TS/Python 为主，**未覆盖 taskBill Go**）。
- `codegraph_explore` MCP：当前会话不可用。
- 设计依据：`handleTransactionsList`、`BillingTransactionsTable.vue`、`_wip_aside/transaction_change_enrich.go` 源码。

## 🐍 Python 新增接口

无。扩展既有 Go `taskBill` GET。
