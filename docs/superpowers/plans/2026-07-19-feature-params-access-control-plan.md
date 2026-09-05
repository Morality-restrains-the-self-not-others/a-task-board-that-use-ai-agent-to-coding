# Feature-Params 访问控制 — 实施计划

## 任务

- [x] 1. 新增 `FeatureParamsAccessAudit` 模型 + migration 0061
- [x] 2. 新增 `projects/services/feature_params_access.py`（resolve / redact / audit）
- [x] 3. 改造 `manage_feature_params` GET/POST 接入门禁与审计
- [x] 4. 改造 `manage_workspace_feature_params` GET/POST 接入门禁与审计
- [x] 5. 更新 Django 测试 `test_manage_feature_params.py` (+ workspace)
- [x] 6. 前端：设置页带 Access-Context；任务详情/预算走 `?view=summary`
- [x] 7. 更新前端单测；必要时 collectstatic
- [x] 8. intents + value-stream 测试点对齐
- [x] 9. Review + PR → https://github.com/task2money/task2app/pull/44

## 验收命令

```bash
cd task2app/Saas_project && python -m pytest tests/test_manage_feature_params.py tests/test_workspace_feature_params_api.py -q
cd task2app/front_project/app && npm test -- --run src/composables/taskDetail/taskDetailLayerGraphModelOptions.test.js
```
