# 容器令牌初始化 DDD 设计（AccessToken / RefreshToken）

## 1) 限界上下文

- **容器令牌生命周期上下文**：负责 `bootstrap access`、`exchange refresh`、`refresh access` 的业务规则与状态机。
- **任务运行配置上下文**：负责实例配置（平台、地域、网络、SSH 等），并持有令牌会话的持久化承载。
- **容器可达性上下文**：消费当前有效 access token，处理 `register-reachability / heartbeat / status-push` 等回调校验。

> 本次领域建模聚焦第一个上下文，并通过仓储接口与后两个上下文解耦。

## 2) 实体与值对象

### 实体（聚合根）

- `ContainerTokenSession`
  - 标识：`TaskScope(tenant_id, workspace_id, task_id, comment_id)`（ADR-0005：评论级；`comment_id` 签发必填）
  - 状态：`current_access_token`、`current_refresh_token`、`access_token_expires_at`、`business_api_endpoint`
  - 关键行为：
    - `bootstrap(...)`
    - `exchange_refresh(...)`
    - `refresh_access(...)`
    - `is_access_token_valid(...)`

### 值对象

- `TaskScope`
- `AccessToken`
- `RefreshToken`
- `BusinessApiEndpoint`

## 3) 聚合与一致性边界

- **聚合：ContainerTokenSessionAggregate**
  - **聚合根：`ContainerTokenSession`**
  - 一致性规则：
    - 初始化时必须清空 refresh token，并签发新的 access token 与过期时间。
    - `exchange-refresh` 成功后必须立即作废 access token（置空）并写入 refresh token。
    - `refresh-access` 成功后必须覆盖当前 access token 与过期时间。
    - 同一 `TaskScope`（含 `comment_id`）任一时刻只能有一组有效 access token。
    - 不同 `comment_id` 互不共享令牌行；存量 `comment_id=''` 视为任务级遗留。

## 4) 领域服务

- `ContainerTokenLifecycleService`
  - `bootstrap_access_token(...)`
  - `exchange_refresh_token(...)`
  - `refresh_access_token(...)`

服务通过构造函数注入仓储与 token 工厂（随机 token 生成器），不依赖 ORM/HTTP/Kafka。

## 5) 仓储接口

- `ContainerTokenSessionRepository`
  - `find_by_scope(...)`
  - `find_by_access_token(...)`
  - `find_by_refresh_token(...)`
  - `save(...)`

> 基础设施层可由 `CloudServerConfig` 适配实现该接口，但不应把 ORM 类型暴露到领域层。

## 6) 领域事件

- `AccessTokenBootstrapped`
- `RefreshTokenExchanged`
- `AccessTokenRefreshed`

事件用于连接上游审计、SSE 通知、运维可观测性，不把基础设施细节耦合到领域对象。

## 7) 文件落点

已生成到 `Saas_project/cloud/domain/`：

- `entities/container_token_session.py`
- `value_objects/task_scope.py`
- `value_objects/access_token.py`
- `value_objects/refresh_token.py`
- `value_objects/business_api_endpoint.py`
- `repositories/container_token_session_repository.py`
- `services/container_token_lifecycle_service.py`
- `events/access_token_bootstrapped.py`
- `events/refresh_token_exchanged.py`
- `events/access_token_refreshed.py`

## 8) 迁移建议（从现有服务到领域层）

1. 在基础设施层新增 `CloudServerConfigTokenSessionRepository`，适配 `ContainerTokenSessionRepository`。
2. 让 `server_config_container_tokens.upsert_bootstrap_container_tokens` 改为调用 `bootstrap_access_token`。
3. 让 `container_runtime_token_views.exchange/refresh` 改为调用 `exchange_refresh_token` 与 `refresh_access_token`。
4. 保持外部 API 行为不变，仅把业务规则收敛到领域层。

