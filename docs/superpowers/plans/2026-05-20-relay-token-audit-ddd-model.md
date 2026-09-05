# DDD Model: relayToTrae Token 审计

> 来源：`docs/superpowers/specs/2026-05-20-relay-token-audit-events-design.md`

## 1) 限界上下文

- **Container Token Lifecycle**：管理 bootstrap / exchange-refresh / refresh-access 令牌生命周期。
- **Relay Token Observability**：记录 relay 使用与 status-push 成功/失败审计事实。

## 2) 实体与值对象

- **实体**
  - `ContainerTokenSession`（现有聚合根）
  - `ContainerTokenAuditEvent`（新增审计事件实体，append-only）
- **值对象**
  - `AccessToken` / `RefreshToken` / `BusinessApiEndpoint` / `TaskScope`（现有）
  - `TokenDigest`（新增，封装 sha256 + suffix）

## 3) 聚合与聚合根

- **Token Session Aggregate**
  - 聚合根：`ContainerTokenSession`
  - 保证换票与刷新时的访问令牌不变式
- **Token Audit Aggregate**
  - 聚合根：`ContainerTokenAuditEvent`
  - 只追加，不更新/删除（清理任务属于运维边界）

## 4) 领域服务

- `ContainerTokenLifecycleService`（现有）
- `ContainerTokenAuditService`（新增，负责组装与追加审计实体）

## 5) 仓储接口

- `ContainerTokenSessionRepository`（现有）
- `ContainerTokenAuditEventRepository`（新增，append + query by task）

## 6) 领域事件（过去式）

- 既有：
  - `AccessTokenBootstrapped`
  - `RefreshTokenExchanged`
  - `AccessTokenRefreshed`
- 新增：
  - `RelayRegistered`
  - `RelayStarted`
  - `StatusPushSucceeded`
  - `StatusPushRejected`

## 7) DDD Hard Gate 自检

- [x] 领域模型位于 `cloud/domain/`
- [x] 领域层无 ORM / requests / 云 SDK 导入
- [x] 仓储接口使用 ABC，实现在 `cloud/infrastructure/persistence/`
- [x] 审计相关领域事件均为过去式命名
- [x] 领域服务通过仓储接口注入依赖

