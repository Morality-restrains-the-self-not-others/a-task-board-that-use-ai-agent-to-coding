# 实施计划 — Token 存储迁移至 Go taskCredentialService

> 输入: 价值流 `2026-07-01-token-storage-go-migration-value-stream.md`、DDD `2026-07-01-token-storage-go-migration-ddd.md`
> 总任务数: 18 | 预计改动文件: 12

---

## Increment 1: Go exchange-refresh + refresh-access (Core)

### T1.1 新增 `ErrTokenExchangeAlreadyDone` 错误码
- **文件**: `taskCredentialService/domain/entities.go`
- **变更**: 在 `var (...)` 块中新增一行
- **验证**: `go build ./...` 通过

### T1.2 实现 `TokenService.ExchangeRefresh()`
- **文件**: `taskCredentialService/application/services.go`
- **变更**: 新增方法，约 35 行
  - 调 `ValidateToken(accessToken, scope)` → 验证
  - 检查 `token.ContainerRefreshToken != ""` → `ErrTokenExchangeAlreadyDone`
  - 生成 `refreshToken` → `s.tokenRepo.UpdateRefreshToken(token.ID, refreshToken)`
  - 作废 access → `s.tokenRepo.UpdateAccessToken(token.ID, "", "")`
  - 调用 `s.recordAudit(token, "exchange_refresh", "", nil)`
- **验证**: Go 测试通过

### T1.3 实现 `TokenService.RefreshAccess()`
- **文件**: `taskCredentialService/application/services.go`
- **变更**: 新增方法，约 30 行
  - 按 refresh_token 查 `tokenRepo.FindByRefreshToken`（需新增 repo 方法，或走现有 FindByTaskID + 比对）
  - scope 校验
  - 生成新 access → `tokenRepo.UpdateAccessToken`
  - 审计
- **验证**: Go 测试通过

### T1.4 handler 新增 `exchange-refresh` case
- **文件**: `taskCredentialService/interfaces/handlers.go`
- **变更**: 在 `handleContainerAPI` switch 中新增 case，约 40 行
  - 解析 `access_token` + `business_api_endpoint` from body
  - 调 `h.svc.Token.ExchangeRefresh()`
  - 返回 `{"refresh_token": ..., "task_id": ...}`
  - 错误映射: `TOKEN_NOT_FOUND`→401, `TOKEN_EXPIRED`→401, `TOKEN_EXCHANGE_ALREADY_DONE`→403
- **验证**: curl 测试

### T1.5 handler 新增 `refresh-access` case
- **文件**: `taskCredentialService/interfaces/handlers.go`
- **变更**: 同上模式，调 `h.svc.Token.RefreshAccess()`，返回 `{"access_token": ..., "expires_at": ...}`
- **验证**: curl 测试

### T1.6 Repository 新增 `FindByRefreshToken` 方法
- **文件**: `taskCredentialService/ports/repositories.go` + `taskCredentialService/infrastructure/sqlite_tokens.go`
- **变更**: 端口接口新增 `FindByRefreshToken(refreshToken string) (*domain.ContainerToken, error)`，SQLite 适配器实现
- **验证**: 现有测试通过

### T1.7 Repository 新增 `UpdateBusinessAPIEndpoint` 或扩展现有 Update
- **文件**: `taskCredentialService/ports/repositories.go` + `taskCredentialService/infrastructure/sqlite_tokens.go`
- **变更**: 端口接口新增 `UpdateBusinessAPIEndpoint(id, endpoint string) error`
- **验证**: `go vet ./...`

---

## Increment 2: taskAgentSupport 路由分叉 (Essential Support)

### T2.1 配置文件新增 `TaskCredentialServiceURL`
- **文件**: `taskAgentSupport/src/config.go`
- **变更**: 在 `serviceConfig` struct 新增 `TaskCredentialServiceURL string`，默认 `http://127.0.0.1:8015`
- **验证**: `go build ./...`

### T2.2 实现 `forwardToCredentialService()`
- **文件**: `taskAgentSupport/src/django_client.go`（或新建 `credential_client.go`）
- **变更**: 新增函数，约 40 行
  - 构建 URL: `{cfg.TaskCredentialServiceURL}/api/tenant/{tid}/workspace/{wid}/task/{tk}/cloud/server-container-token/{action}/`
  - 转发 body，携带 `X-Trace-Id` header
  - 返回 `(statusCode, respBody, error)`
- **验证**: 单元测试

### T2.3 `handleCloudInbound` 路由分叉
- **文件**: `taskAgentSupport/src/handlers.go`
- **变更**: 在 `handleCloudInbound` 中，对 `exchange-refresh` / `refresh-access` action 调用 `forwardToCredentialService` 而非 `forwardToDjango`
- **验证**: `go test ./...`

---

## Increment 3: Go /v1/token/validate 端点 (Essential Support)

### T3.1 实现 `handleValidateToken`
- **文件**: `taskCredentialService/interfaces/handlers.go`
- **变更**: 新增 handler，约 30 行
  - 解析 `{"access_token": "..."}`
  - 调 `h.svc.Token.ValidateToken(accessToken, scope)` — 用宽松 scope（不强制校验 tenant/workspace/task）
  - 或者新增 `ValidateTokenAnyScope(accessToken)` 方法
  - 返回 `{"valid": true, "company_id": ..., "workspace_id": ..., "task_id": ...}`
- **验证**: curl `POST /v1/token/validate`

### T3.2 注册 `/v1/token/validate` 路由 + internal secret 保护
- **文件**: `taskCredentialService/interfaces/handlers.go`
- **变更**: 在 `RegisterRoutes` 中新增 `mux.HandleFunc("/v1/token/validate", h.handleValidateToken)`
- **验证**: 不带 secret → 403；带 secret → 200

---

## Increment 4: Django 下游 view 切换 (Core)

### T4.1 新建 `TokenValidationPort` 端口接口
- **文件**: `task2app/Saas_project/cloud/domain/ports/external/token_validation_port.py` (NEW)
- **变更**: ABC，定义 `validate(access_token: AccessToken) -> Optional[TaskScope]`
- **验证**: `import` 无循环依赖

### T4.2 新建 `GoTokenValidator` 适配器
- **文件**: `task2app/Saas_project/cloud/infrastructure/adapters/go_token_validator.py` (NEW)
- **变更**: 实现 `TokenValidationPort`，调 `POST http://127.0.0.1:8015/v1/token/validate`
  - 携带 `X-TaskAgentSupport-Internal-Secret`
  - 返回 `TaskScope` 或 `None`
- **验证**: 单元测试（mock Go 响应）

### T4.3 改造 `_resolve_cfg_by_access_token_for_callback`
- **文件**: `task2app/Saas_project/cloud/views/container_runtime_token_views.py`
- **变更**: 
  - `CloudServerConfig.objects.filter(container_access_token=access).first()` 
  - → `token_validator.validate(AccessToken(access))` → scope → `CloudServerConfig.objects.filter(company_id=..., workspace_id=..., task_id=...)`
- **影响范围**: heartbeat, register-reachability 共用此函数
- **验证**: 现有测试通过 + 新增 Go 不可达场景测试

### T4.4 改造其他 5 个下游 view
- **文件**: 
  - `container_feature_params_views.py` (1处)
  - `container_git_clone_progress_views.py` (3处)
  - `container_layer_github_oauth_views.py` (1处)
  - `container_task_detail_views.py` (2处)
  - `task_model_budget_usage_views.py` (1处)
- **变更**: 每个文件的 `CloudServerConfig.objects.filter(container_access_token=access).first()` → 调用 `token_validator.validate()` + scope 查询
- **验证**: 每个文件的对应测试通过

---

## Increment 5: Django internal_dispatch 清理 (Cleanup)

### T5.1 移除 `exchange-refresh`/`refresh-access` 映射
- **文件**: `task2app/Saas_project/cloud/task_agent_support/internal_dispatch.py`
- **变更**: 从 `_VIEW_BY_ACTION` dict 中移除 `exchange-refresh` 和 `refresh-access` 条目
- **验证**: 请求 `exchange-refresh` → 404

### T5.2 标记废弃函数
- **文件**: `task2app/Saas_project/cloud/views/container_runtime_token_views.py`
- **变更**: 在 `exchange_server_container_refresh_token` 和 `refresh_server_container_access_token` 函数上添加 deprecation 注释
- **验证**: import 无 broken reference

---

## Increment 6: 测试 (Essential Support)

### T6.1 Go 单元测试
- **文件**: `taskCredentialService/token_test.go`
- **新增用例**:
  - `TestTokenService_ExchangeRefresh_Success` — 正常换取
  - `TestTokenService_ExchangeRefresh_AlreadyDone` — 重复换取 → `TOKEN_EXCHANGE_ALREADY_DONE`
  - `TestTokenService_ExchangeRefresh_InvalidToken` — 无效 token → `TOKEN_NOT_FOUND`
  - `TestTokenService_RefreshAccess_Success` — 正常刷新
  - `TestTokenService_RefreshAccess_InvalidRefreshToken` — 无效 refresh → 401
  - `TestValidateToken_Valid` — 有效 token 返回 scope
  - `TestValidateToken_Expired` — 过期 token → 401
- **验证**: `go test ./... -count=1`

### T6.2 Django 集成测试更新
- **文件**: `task2app/Saas_project/tests/test_container_runtime_tokens.py`
- **变更**: mock `GoTokenValidator.validate` 替代 `CloudServerConfig.objects.filter(container_access_token=...)`
- **新增用例**: `test_exchange_refresh_go_unreachable_503` — Go 不可达时返回 503
- **验证**: `pytest tests/test_container_runtime_tokens.py -x`

### T6.3 taskAgentSupport 路由测试
- **文件**: `taskAgentSupport/src/handlers_test.go`
- **新增用例**: `exchange-refresh` / `refresh-access` action 路由到 `forwardToCredentialService`
- **验证**: `go test ./...`

---

## 任务依赖图

```
T1.1 (ErrTokenExchangeAlreadyDone)
  └─ T1.2 (ExchangeRefresh)
       └─ T1.4 (handler exchange-refresh case)
            └─ T2.3 (路由分叉) ── T5.1 (移除映射)
                 └─ T6.3 (路由测试)
T1.6 (FindByRefreshToken)
  └─ T1.3 (RefreshAccess)
       └─ T1.5 (handler refresh-access case)
            └─ T2.3 (同上)
T3.1 (validate handler) ── T3.2 (路由注册)
  └─ T4.2 (GoTokenValidator)
       └─ T4.3 (_resolve_cfg 改造)
            └─ T4.4 (5个下游view)
                 └─ T5.2 (标记废弃) ── T6.2 (集成测试)
T1.2 ── T6.1 (Go 单元测试)
```

## 执行顺序

```
Phase 1 (并行): T1.1, T1.6, T1.7, T2.1, T3.1
Phase 2:        T1.2, T1.3
Phase 3:        T1.4, T1.5, T3.2
Phase 4:        T2.2, T2.3, T4.1
Phase 5:        T4.2, T4.3
Phase 6:        T4.4, T5.1, T5.2
Phase 7:        T6.1, T6.2, T6.3
```
