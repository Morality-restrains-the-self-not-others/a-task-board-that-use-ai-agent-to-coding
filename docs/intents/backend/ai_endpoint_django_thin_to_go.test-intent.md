# AI endpoint Django thin 迁 Go — 测试意图

## 用例

1. `go test` taskCloudService `-run 'ResolveProvider|AIEndpoint'`
2. `go test` taskAIEndPoint `-run 'ValidateProxy|CloudResolve|CloudBudget'`
3. Django `test_django_task_ai_endpoint_internal_urls_gone` → 全部 404
4. live：Cloud resolve-route 200；Django task-ai-endpoint 404

## 命令

```bash
cd /tmp/ram-work/taskCloudService && go test ./src/ -count=1 -run 'ResolveProvider|AIEndpoint|Budget'
cd /tmp/ram-work/taskAIEndPoint && go test ./src/ -count=1
cd /tmp/ram-work/task2app && ./activate_env.sh run -- bash -lc \
  'cd Saas_project && python -m pytest tests/test_task_ai_endpoint_internal.py -q'
```
