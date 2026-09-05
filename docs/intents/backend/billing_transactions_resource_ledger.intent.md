# 功能意图：交易列表返回资源变动与瞬时账目

## 背景与目标

租户交易列表无法当流水账阅读：配额行看不出标的，现金剩余以分裸露且无 before→after。目标是在既有 GET 上返回 `change_display` / `resource_changes` / `ledger_snapshot`。

## 范围与边界

- 范围内：`handleTransactionsList`（filtered 与否均 enrich）
- 范围外：新 endpoint、新表、GitLab 区域配额历史回放、写路径落库快照

## 约束与风险

- 向后兼容：新字段可选
- 筛选下列表回放任务帖剩余必须用未过滤历史，避免瞬时剩余被滤掉的行打乱
- 同一秒多笔 `admin_grant` 与 grant 行时间对齐仍可能串单（沿用秒级匹配；OPT 改为 transaction_id 关联）

## 验收标准

1. 后台赠送行 `change_display` 含资源名与数量（如 `任务帖 +10 帖`），且有 `resource_changes[]`
2. 现金消耗行 `ledger_snapshot.display` 为「余额 a.aa → b.bb 元」（分÷100）
3. 配额消耗（`amount_points=0` 且 `usage_amount>0`）`change_display` 非 `+0.00 元`
4. OpenAPI `BillingTransaction` 含上述可选字段

## 实施计划

1. 将 `_wip_aside` enrich 迁入 `taskBill/src`
2. 增加 `attachLedgerSnapshots` + 任务帖剩余回放
3. 单测 + OpenAPI

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 查询交易流水账 | — | — | GET list | — | 纯查询，无对应事件 |
