# [运行时] 交易记录「计费单元」列显示「-」

## 现象

租户控制台 `…/billing/transactions/` 表格「计费单元」列对消费流水也显示 `-`（HTML 为 `<td …>-</td>`），即使用量描述已写明「任务帖创建 / 任务服务器启动」等。

## 环境与上下文

- 前端：`BillingTransactions.vue` → `transaction.billing_unit?.name || '-'`
- API：`GET /api/tenant/{id}/billing/transactions/list_filtered/`（taskBill）
- 用量页同样依赖嵌套对象：`BillingUsage.vue` → `billing_unit?.name` / `billing_unit?.unit`

## 根因

taskBill `handleTransactionsList` / `handleUsagesList` 把 `billing_unit` **序列化成 ID 字符串**（`formatID(unitID)`），而前端契约是 **对象** `{ name, unit, … }`。

对字符串取 `.name` 为 `undefined`，回落为 `-`。充值流水本身无 `billing_unit_id`，显示 `-` 是预期。

## 修复

- 列表接口批量加载 `billing_unit` 表，将 `billing_unit` 展开为嵌套对象（含 `id`/`name`/`unit`/`unit_type`/`price_points`）
- 单测：`handlers_billing_unit_nested_test.go`
- 重启 `taskBill` 后本地 API：88 条消费均有 `name`（如「智能体任务」「任务帖创建」）

## 后续（充值无 unit）

充值写入曾不设 `billing_unit_id`，故展开后仍无对象。已另见 `19_billing_recharge_missing_billing_unit.md`：种子 `recharge` 单元 + `creditRecharge` 写入 + 历史回填。

## 预防

- 前后端字段契约以**嵌套展示对象**为准时，禁止把 FK 仅当标量 ID 返回
- 改序列化后用「前端表达式能读到的形态」做 API 断言（对象 + `name`），勿只断言字段存在
