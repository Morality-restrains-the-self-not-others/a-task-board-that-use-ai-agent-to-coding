# Value Stream: Container Token 存储完整迁移至 Go taskCredentialService

> Derived from design: brainstorming session (token exchange failed: HTTP 401 TOKEN_ACCESS_INVALID)

## Value Summary

容器运行时 token 的 `exchange-refresh` / `refresh-access` 生命周期操作从 Django `CloudServerConfig` 表迁移至 Go `taskCredentialService` 的 `container_tokens` 表，Go 成为 token 存储与操作的唯一数据源（SSOT），消除 relayToTrae 启动时 Django↔Go 数据不一致导致的 401 错误。

## Related Value Streams

- **`relay-precheck-ssot`** (`2026-07-01-relay-precheck-ssot-value-stream.md`): **extension** — 该流已将 token-init 迁至 Go。本流是第二步，将 token 生命周期操作（exchange/refresh）也迁至 Go，完成架构归一化。
- **`relay-register-token-exchange-guard`** (`conf/value-stream.yaml`): **modification** — TEIP guard 逻辑已部分迁移至 Go，本流将其 `exchange-refresh-then-refresh-access` 步骤从 Django `CloudServerConfig` 字段改为 Go `container_tokens` 字段。
- **`task-container-gateway`** (`2026-05-31-task-container-gateway-value-stream.md`): **alignment** — taskAgentSupport 路由分叉需要该 gateway 就绪。

## End-to-End Flow

```
[容器启动 / go_relayToTrae]
  │
  ├── POST .../exchange-refresh/  ──→  [taskAgentSupport 路由] ──→  [Go taskCredentialService]
  │     │                                                              │
  │     │  验证 access_token ──→ Go container_tokens 表                 │
  │     │  生成 refresh_token ──→ 写入 Go container_tokens 表           │
  │     │  作废 access_token  ──→ 清空 Go container_tokens 表           │
  │     │  审计 ──→ Go token_audit_events 表                            │
  │     │  返回 refresh_token                                          │
  │     │                                                              │
  │     └── [Django SSE 副作用] ←── Go 回调 Django 或容器自行触发
  │
  ├── POST .../refresh-access/  ──→  [taskAgentSupport 路由] ──→  [Go taskCredentialService]
  │     │                                                              │
  │     │  验证 refresh_token ──→ Go container_tokens 表               │
  │     │  生成新 access_token ──→ 写入 Go container_tokens 表          │
  │     │  返回新 access_token                                         │
  │
  └── [Django 下游 view: heartbeat, task-detail, ...]
        │
        └── POST /v1/token/validate ──→ [Go taskCredentialService]
              │
              验证 access_token ──→ 返回 {valid, company_id, workspace_id, task_id}
              │
              Django 用 scope 查 CloudServerConfig ──→ 继续业务逻辑
```

## Value Increments

### Increment 1: Go taskCredentialService — exchange-refresh + refresh-access (Core)
**Value to user:** relayToTrae 启动时的 token exchange 不再因双数据库不一致而 401 失败。
**Scope:** 
- `domain/entities.go`: 新增 `ErrTokenExchangeAlreadyDone` 错误码
- `application/services.go`: 新增 `ExchangeRefresh()`, `RefreshAccess()` 方法
- `interfaces/handlers.go`: 新增 `exchange-refresh`, `refresh-access` case
- `infrastructure/sqlite_tokens.go`: 新增 `UpdateBusinessAPIEndpoint()` 或扩展现有 update 方法
**Depends on:** Go taskCredentialService 已有 `IssueToken`, `ValidateToken`, `UpdateAccessToken`, `UpdateRefreshToken`

### Increment 2: taskAgentSupport 路由分叉 (Essential Support)
**Value to user:** 外部容器回调请求正确路由到 Go 而非 Django。
**Scope:**
- `taskAgentSupport/src/config.go`: 新增 `TaskCredentialServiceURL` 配置
- `taskAgentSupport/src/django_client.go`: 新增 `forwardToCredentialService()` 函数
- `taskAgentSupport/src/handlers.go`: `exchange-refresh`/`refresh-access` 路由到 Go
**Depends on:** Increment 1

### Increment 3: Go `/v1/token/validate` 端点 (Essential Support)
**Value to user:** Django 下游 view 能验证 token 而不直接读数据库。
**Scope:**
- `interfaces/handlers.go`: 新增 `POST /v1/token/validate` handler
- `application/services.go`: 复用 `ValidateToken()` 方法
- 内部 secret 校验 middleware
**Depends on:** Increment 1

### Increment 4: Django 下游 view 切换到 Go validate (Core)
**Value to user:** 容器运行时回调（heartbeat, task-detail, git-clone-progress 等）正确验证 token。
**Scope:**
- `container_runtime_token_views.py`: `_resolve_cfg_by_access_token_for_callback` → 调用 Go validate + scope 查 CloudServerConfig
- `container_feature_params_views.py`: 同上
- `container_git_clone_progress_views.py`: 同上（3处）
- `container_layer_github_oauth_views.py`: 同上
- `container_task_detail_views.py`: 同上（2处）
- `task_model_budget_usage_views.py`: 同上
- `relay_to_trae_proxy.py`: 新增 `_validate_token_via_go()` 辅助函数
**Depends on:** Increment 3

### Increment 5: Django internal_dispatch 清理 (Cleanup)
**Value to user:** 代码库清理，消除死代码。
**Scope:**
- `internal_dispatch.py`: 移除 `exchange-refresh`/`refresh-access` 映射
- `container_runtime_token_views.py`: 标记 `exchange_server_container_refresh_token`/`refresh_server_container_access_token` 为 deprecated
**Depends on:** Increment 1-4

### Increment 6: 测试更新与回归保护 (Essential Support)
**Value to user:** 行为长期稳定，重构不出回归。
**Scope:**
- 更新 `test_container_runtime_tokens.py`: mock Go HTTP 调用替代 Django DB 断言
- 更新 `test_relay_register_token_exchange_guard.py`: 字段引用更新
- 新增 Go 侧单元测试: `TestTokenService_ExchangeRefresh`, `TestTokenService_RefreshAccess`
- 新增 `test_go_token_validate.py`: Django 调用 Go validate 的集成测试
**Depends on:** Increment 1-5
