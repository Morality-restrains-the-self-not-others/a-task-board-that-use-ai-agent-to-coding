# 测试意图：交易列表资源变动与瞬时账目

## 测试目标

证明列表 API 在赠送、现金消耗、配额消耗三类行上返回可读流水账字段。

## 测试分层

- Go 单元：`taskBill/src/transaction_change_enrich_test.go`
- Go 单元：`taskBill/src/transaction_ledger_snapshot_test.go`
- OpenAPI：`taskBill/src/openapi.yaml` + 既有 schema 测试不回归

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | admin_grant 任务帖 +10 | `change_display=任务帖 +10 帖`，`resource_changes[0].resource_type=task_post` |
| T2 | 遗留描述「管理员后台赠送资源」+ 同时刻 grant | 描述与 `change_display` 被 enrich |
| T3 | `quota_consumption` amount=0 usage=1 | `change_display` 含数量与单元名，非 ±0.00 元 |
| T4 | consumption amount=30 before=100 after=70 | `ledger_snapshot.display` 含 `1.00` 与 `0.70` |
| T5 | 两笔任务帖消耗后列表 | 较新行 `remaining_after` 小于较旧行（回放方向正确） |

## 数据与环境

`setupMySQLTestDB`；独立 tenant_id，避免并行污染。

## 通过标准

上述用例全绿；既有 `handlers_billing_unit_nested_test.go` 仍通过。

## 业务意图 → 事件对照（测试）

纯查询，不断言 MQ 投递。
