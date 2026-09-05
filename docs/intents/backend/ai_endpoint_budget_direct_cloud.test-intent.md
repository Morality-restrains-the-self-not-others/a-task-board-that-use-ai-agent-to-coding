# AI endpoint Budget 直打 Cloud — 测试意图

## 用例

1. `go test ./src/ -run CloudBudget`：reserve 402 + commit 命中 mock Cloud
2. Django：`test_django_budget_thin_urls_gone` → 404
3. Django：`test_resolve_route` / `test_validate_proxy_token` 仍通过
4. live（可选）：`POST :8018/api/internal/budget/reserve-or-deny/` 与 Django budget URL 404

## 命令

```bash
cd /tmp/ram-work/taskAIEndPoint && go test ./src/ -count=1 -run 'CloudBudget|Parse'
cd /tmp/ram-work/task2app && ./activate_env.sh run -- bash -lc \
  'cd Saas_project && python -m pytest tests/test_task_ai_endpoint_internal.py -q'
```
