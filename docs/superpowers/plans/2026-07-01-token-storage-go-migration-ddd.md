# DDD 领域模型变更 — Token 存储迁移至 Go

> 输入: 价值流 `2026-07-01-token-storage-go-migration-value-stream.md`、NFR `2026-07-01-token-storage-go-migration-nfr-clarification.md`
> 性质: **增量修改**（非新建），现有领域模型已完备

## 当前领域模型状态

### Go `taskCredentialService` (Token 限界上下文)

```
domain/
├── entities.go           — ContainerToken (聚合根), TaskScope (VO), AccessToken (VO), RefreshToken (VO)
├── events.go             — TokenIssued, TokenExchanged, TokenRefreshed, TokenAuditRecorded
ports/
├── repositories.go       — ContainerTokenRepository, TokenAuditEventRepository, BusinessDataRepository, GitoauthClient
application/
├── services.go           — TokenService{IssueToken, ValidateToken}, CredentialService, TaskDetailService
infrastructure/
├── sqlite_tokens.go      — SQLiteTokenRepository (实现 ContainerTokenRepository)
├── composition.go        — Compose() DI 组装
interfaces/
├── handlers.go           — handleContainerAPI (repo-clone-credentials, task-detail)
```

### Django `cloud` (容器运行时限界上下文)

```
domain/
├── entities/container_token_session.py         — ContainerTokenSession (聚合根)
├── value_objects/access_token.py, refresh_token.py, task_scope.py
├── repositories/container_token_session_repository.py  — ContainerTokenSessionRepository (端口)
├── services/container_token_lifecycle_service.py       — bootstrap_access_token, exchange_refresh_token, refresh_access_token
infrastructure/
├── persistence/django_container_token_session_repository.py  — DjangoContainerTokenSessionRepository (适配器)
```

## 变更内容

### 1. Go `application/services.go` — 新增领域服务方法

```go
// ExchangeRefresh: access_token → refresh_token（一次性操作）
func (s *TokenService) ExchangeRefresh(accessToken, businessAPIEndpoint string, scope TaskScope) (string, error) {
    // 1. ValidateToken → 验证 access_token 非空、scope 匹配、未过期
    // 2. 检查 ContainerRefreshToken == "" → 防止重复交换
    // 3. 生成新 refresh_token → UpdateRefreshToken
    // 4. 作废 access_token → UpdateAccessToken("", "")
    // 5. recordAudit("exchange_refresh")
    // 6. 返回 refresh_token
}

// RefreshAccess: refresh_token → new_access_token
func (s *TokenService) RefreshAccess(refreshToken string, scope TaskScope) (string, string, error) {
    // 1. FindByRefreshToken → 验证 refresh_token
    // 2. scope 校验
    // 3. 生成新 access_token → UpdateAccessToken
    // 4. recordAudit("refresh_access")
    // 5. 返回 new_access_token, expires_at
}
```

### 2. Go `domain/entities.go` — 新增错误码

```go
var (
    // ... existing errors ...
    ErrTokenExchangeAlreadyDone = &DomainError{Code: "TOKEN_EXCHANGE_ALREADY_DONE", Message: "预埋 AccessToken 仅可用于首次换取 RefreshToken"}
)
```

### 3. Go `interfaces/handlers.go` — 新增 handler cases

```go
// handleContainerAPI switch 新增:
case "exchange-refresh":
    h.handleExchangeRefresh(w, r, tenantID, workspaceID, taskID)
case "refresh-access":
    h.handleRefreshAccess(w, r, tenantID, workspaceID, taskID)

// 新增独立端点:
mux.HandleFunc("/v1/token/validate", h.handleValidateToken)  // POST, internal secret protected
```

### 4. Django — 端口接口新增

Django 侧新增外部服务端口（调用 Go validate-token）：

```python
# cloud/domain/ports/external/token_validation_port.py (NEW)
class TokenValidationPort(ABC):
    """验证容器 access_token 的端口接口。实现者调用 Go /v1/token/validate。"""
    @abstractmethod
    def validate(self, access_token: AccessToken) -> Optional[TaskScope]:
        """验证 token 并返回 scope，无效/过期返回 None"""
```

适配器实现:

```python
# cloud/infrastructure/adapters/go_token_validator.py (NEW)
class GoTokenValidator(TokenValidationPort):
    """调用 Go taskCredentialService /v1/token/validate 的 HTTP 适配器"""
    def validate(self, access_token: AccessToken) -> Optional[TaskScope]:
        resp = _credential_service_post(
            f"{base_url}/v1/token/validate",
            json={"access_token": access_token.value},
            timeout=5,
        )
        if resp.status_code == 200:
            return TaskScope(tenant_id=..., workspace_id=..., task_id=...)
        return None
```

### 5. Django — 下游 view 修改

`_resolve_cfg_by_access_token_for_callback` 改为两步:
```python
# 原: cfg = CloudServerConfig.objects.filter(container_access_token=access).first()
# 新:
scope = token_validator.validate(AccessToken(access))  # → Go RPC
if scope is None:
    return error_response(401, TOKEN_ACCESS_INVALID)
cfg = CloudServerConfig.objects.filter(
    company_id=scope.tenant_id,
    workspace_id=scope.workspace_id,
    task_id=scope.task_id,
).first()
```

### 6. Django — 废弃方法

`container_token_lifecycle_service.py`:
- `exchange_refresh_token()` → 不再调用（Go 接管）
- `refresh_access_token()` → 不再调用（Go 接管）
- `bootstrap_access_token()` → 保留（upsert_bootstrap_container_tokens 仍用）

## 依赖反转验证 ✅

```
Go 侧:
  interfaces/handlers.go  →  application/TokenService  →  domain/ContainerToken
                                                              ↑ (实现)
                                        infrastructure/SQLiteTokenRepository

Django 侧:
  views/container_runtime_token_views.py
         → domain/ports/external/TokenValidationPort (ABC)
                  ↑ (实现)
         infrastructure/adapters/GoTokenValidator (HTTP 适配器)

切换验证: 如果未来不用 HTTP 调用 Go 而是共享 SQLite → 只需新建 SQLiteTokenValidator 适配器
```

## 自检

- [x] 领域层无 ORM 导入 — Go/Django 均使用现有纯领域模型
- [x] 端口接口由领域层定义 — `TokenValidationPort` 在 `domain/ports/` 下
- [x] 基础设施适配器实现端口 — `GoTokenValidator` 实现 `TokenValidationPort`
- [x] 聚合不变 — `ContainerToken` (Go)、`ContainerTokenSession` (Django) 聚合边界不变
- [x] 新增操作在现有聚合上 — `ExchangeRefresh`/`RefreshAccess` 在 `TokenService` 上
