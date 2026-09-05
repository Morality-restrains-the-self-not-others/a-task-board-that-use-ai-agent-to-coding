# [运行时] 用量页「用户」列显示公司成员 ID

## 现象

租户控制台 `…/billing/usage/` 表格「用户」列显示 Snowflake ID（如 `850256676127797248`），而非公司成员名称。

## 环境与上下文

- 前端：`BillingUsage.vue` → `record.user_name || record.user_id || '-'`（已优先名称）
- API：`GET /api/tenant/{id}/billing/usages/`（taskBill）
- 对照：`list_filtered` 交易列表已调用 `djangoEnrichTransactions` 补齐 `user_name`

## 根因

`handleUsagesList` 只回传 `user_id`（常为 `company_member_id`），**未**走 Django `enrich-transactions` 解析 `CompanyMember.member_name`。前端无 `user_name` 时回落显示 ID。

## 修复

- `handleUsagesList` 在响应前复用 `djangoEnrichTransactions`（同一 internal 接口按 `user_id`/`company_member_id` 补 `user_name`，并顺带补 `project_name`）
- 单测：`handlers_usages_enrich_user_name_test.go`
- 重启 taskBill 后本地：`user_id=850256676127797248` → `user_name=公司创建者`

## 预防

- 凡列表页「展示名」列（用户/项目/工作空间），API 须与交易列表同一 enrich 路径，禁止只返回裸 ID
- 新增 billing 列表接口时对照 `handleTransactionsList(filtered=true)` 的 enrich 调用
