# Feature-Params 迁 Go — 计划

- [x] 1. Django internal：tenant/workspace/personal upsert + list + access-audit write
- [x] 2. Go：access gate + redact + public handlers（公司/工作空间/个人）
- [x] 3. Go：mount `/api/tenant/.../feature-params` 与 `/api/personal/feature-params-configs`
- [x] 4. taskGateway 高优路由
- [x] 5. Django 公网视图 410 + 测试调整
- [x] 6. ownership YAML 更新
- [x] 7. Go/Django 测试通过 + PR（taskCloudService / task2app / taskGateway）

验收：
```bash
cd taskCloudService/src && go test -count=1 -run FeatureParamsPublic ./...
cd task2app/Saas_project && python3 -m pytest tests/test_manage_feature_params.py tests/test_feature_params_internal_upsert.py -q
```
