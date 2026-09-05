# Implementation Plan: taskGateway Federated Swagger (D1–D3)

> Spec: `docs/superpowers/specs/2026-06-04-taskgateway-federated-swagger-design.md`

## Task 1: conf + routes.yaml docs 契约

- [ ] `conf/task-gateway/config.yaml` 增加 `docs` 块（`enabled: false`）
- [ ] `routes/routes.yaml` 各 upstream 增加 `docs` 或 `docs: false`

## Task 2: routes-to-apisix codegen（D1）

- [ ] `TASK_GATEWAY_DOCS_ENABLED` 环境变量覆盖
- [ ] 生成 docs 路由（portal、ui、schema）
- [ ] 生成 `generated/docs-portal/index.html`
- [ ] `docker-compose.yml` 增加 `docs-portal` nginx 服务

## Task 3: django + gitOauth server URL（D2）

- [ ] django `SWAGGER_SETTINGS['DEFAULT_API_URL']` ← `TASK_GATEWAY_PUBLIC_BASE`
- [ ] gitOauth `SPECTACULAR_SETTINGS['SERVERS']`

## Task 4: tests + value-stream.yaml（D3）

- [x] 扩展 `test_taskgateway_routes_codegen.py`
- [x] `value-stream.yaml` 新增 gateway-docs-* steps
- [x] `routes-apply` 刷新 `apisix.yaml`

## Task 5: taskAuth OpenAPI（D4）

- [x] `taskAuth/src/openapi.yaml`
- [x] `GET /api/schema/`、`GET /api/swagger/`
- [x] `routes.yaml` taskAuth `docs` 启用
- [x] `taskAuth/src/openapi_test.go`

## 验证

```bash
cd task2app/Saas_project && pytest tests/test_taskgateway_routes_codegen.py -q
bash scripts/ci/check_taskgateway_routes.sh
TASK_GATEWAY_DOCS_ENABLED=1 python3 taskGateway/scripts/routes-to-apisix.py
```
