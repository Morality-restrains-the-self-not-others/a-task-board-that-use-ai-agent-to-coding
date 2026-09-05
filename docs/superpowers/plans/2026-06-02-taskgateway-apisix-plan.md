# Implementation Plan: taskGateway（APISIX）G1–G6

> Spec: `docs/superpowers/specs/2026-06-02-taskgateway-apisix-design.md`

## Task 1: conf + taskGateway 骨架

- [x] `conf/task-gateway/config.yaml`
- [x] `taskGateway/docker-compose.yml`、`run.sh`、`routes/routes.yaml`
- [x] `scripts/routes-to-apisix.py` → `apisix/apisix.yaml`
- [x] `runAll.yaml` 增加 `task-gateway` 服务

## Task 2: G2 路由拆分

- [x] routes.yaml：task-auth / git-oauth / task-sse / t-c-gateway 优先规则
- [x] `/api/internal/*` deny 路由

## Task 3: G3 forward-auth + Django

- [x] taskAuth `GET /api/internal/gateway/forward-auth/`
- [x] APISIX forward-auth 插件配置（受保护路由）
- [x] `CustomTokenAuthentication` 信任网关头
- [x] `settings.py`：`TASK_GATEWAY_*`

## Task 4: 前端与 OAuth URL

- [x] `conf/vue/config.yaml` apiBaseUrl → https gateway
- [x] `vite.config.js`（配置 apiBaseUrl 时跳过 /api proxy）
- [x] `GithubAppAuthorizeStartView` 使用 `TASK_GATEWAY_PUBLIC_BASE`
- [x] 本地 provider `redirect_uri` → `https://172.20.10.3:8443`

## Task 5: G4 可观测 + 限流

- [x] `request-id` → `X-Trace-Id`
- [x] `file-logger` → `taskGateway/logs/`
- [x] `limit-req` on `taskauth-login`

## Task 6: G6 CI

- [x] `scripts/ci/check_taskgateway_routes.sh`
- [x] `tests/test_taskgateway_routes_codegen.py` 覆盖 `--check`

## 验证

```bash
cd taskGateway && bash run.sh setup-tls && bash run.sh routes-apply && bash run.sh start
curl -k https://172.20.10.3:8443/api/health/
bash scripts/ci/check_taskgateway_routes.sh
cd task2app/Saas_project && pytest tests/test_taskgateway_routes_codegen.py tests/test_gateway_trust_headers_auth.py -q
```
