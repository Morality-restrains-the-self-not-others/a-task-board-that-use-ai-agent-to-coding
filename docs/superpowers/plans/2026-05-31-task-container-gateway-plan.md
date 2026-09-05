# Plan: taskContainerGateway（Increment 1 薄切片）

> 设计: `docs/superpowers/specs/2026-05-31-task-container-gateway-auth-design.md`  
> 价值流: `docs/superpowers/plans/2026-05-31-task-container-gateway-value-stream.md`

## Tasks

- [x] **T1** Django `cloud/task_container_gateway/internal_auth.py` + secret settings
- [x] **T2** Django `validate-session` + `resolve-container-target` internal views + urls
- [x] **T3** `tests/test_gateway_validate_session.py`（红→绿）
- [x] **T4** Go module `taskContainerGateway/` skeleton + config + health
- [x] **T5** Go `handleContainerCompute` — validate → resolve → forward git-commit
- [x] **T6** Go `handlers_test.go` URL 解析 + mock django
- [x] **T7** `port_config.json` + `runAll.yaml` 登记 :8014
- [x] **T8** taskGateway 路由：`container-layer-*` → taskContainerGateway :8014（~~Inc 1 曾用 Vite proxy~~，2026-07-05 对齐 routes.yaml）
- [x] **T9** `value-stream.yaml` 新流 `task-container-gateway`

## Verify

```bash
cd taskContainerGateway && go test ./...
cd task2app/Saas_project && pytest tests/test_gateway_validate_session.py -q
```
