# 控制台限额 CRUD → Cloud — 测试意图

## 用例

1. Go：`TestWorkspaceBudgetDefaultsCRUD` / `TestTaskModelBudgetsUpsertList`（或等价）
2. Django：`test_workspace_model_budget_defaults` / `test_task_model_budget_override` 经 stub 绿
3. live（可选）：Cloud `workspace-defaults` upsert + list

## 命令

```bash
cd /tmp/ram-work/taskCloudService && go test ./src/ -count=1 -run 'BudgetConfig|WorkspaceBudget|TaskModelBudget'
cd /tmp/ram-work/task2app && ./activate_env.sh run -- bash -lc \
  'cd Saas_project && python -m pytest tests/test_workspace_model_budget_defaults.py tests/test_task_model_budget_override.py -q'
```
