# 实施计划 — 交易记录资源流水账

## 切片 1 — API 展示字段（Red→Green）

- [ ] 迁入 `transaction_change_display.go` / `transaction_change_enrich.go`
- [ ] `handleTransactionsList` 末尾调用 enrich（filtered 前后均可，须在 billing_unit 嵌套之后）
- [ ] 测试 T1–T3：`transaction_change_enrich_test.go`

验证：`go test ./src -count=1 -run 'TestTransactionsListExposesGrantChangeDisplay|TestTransactionsListEnrichesLegacyGenericGrantDescription|TestTransactionsListQuotaConsumptionChangeDisplay'`

## 切片 2 — 瞬时账目快照

- [ ] `attachLedgerSnapshots`：现金 before→after 元
- [ ] 任务帖 `remaining_after` 未过滤回放
- [ ] 测试 T4–T5：`transaction_ledger_snapshot_test.go`
- [ ] OpenAPI `BillingTransaction` 增可选字段

验证：`go test ./src -count=1 -run 'TestTransactionsListLedgerSnapshot|TestTransactionsListTaskPostRemainingReplay'`

## 切片 3 — 前端流水账列

- [ ] `ledgerSnapshotDisplay` helper + 单测 F3
- [ ] `BillingTransactionsTable` 列：变动明细 / 瞬时账目
- [ ] Dashboard 最近交易同步
- [ ] 更新 grantDisplay 测试 F1/F2/F4

验证：vitest 对应文件

## 切片 4 — 意图/价值流/图

- [ ] `docs/intents/INDEX.md`
- [ ] `conf/value-stream.yaml`
- [ ] `docs/flows/value-stream-test-integration.wsd` 测试点

## 事件任务

无发布任务（纯查询）。意图对照表已写例外。

## 可观测性

grant enrich 失败：`slog.WarnContext` event `billing_txn_grant_enrich_failed`；成功：`billing_txn_grant_display_enriched`。
