# saas 遗留 Budget usage 归档

由 `../archive_saas_legacy_usage.sh` 生成 `.sql` / `.csv` 导出；
由 `../drop_saas_legacy_usage.sh --force` 在确认归档后 **DROP** `*_legacy` 表。

- 源表：`projects_task_model_budget_usage`、`projects_task_model_budget_usage_idempotency`
- saas 侧步骤：`RENAME` → `*_legacy` →（可选）`DROP`
- **不**处理 `projects_task_api_key_usage`
- 账本 SSOT：`db/task_budget/task_budget.db`

见意图：[003_budget多副本Postgres评估.intent.md](../../../task2app/docs/intents/engineering/cloud/003_budget多副本Postgres评估.intent.md)
