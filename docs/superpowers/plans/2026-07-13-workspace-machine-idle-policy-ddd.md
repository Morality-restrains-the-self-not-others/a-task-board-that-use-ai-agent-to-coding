# DDD：工作空间机器节点闲置策略

- 日期：2026-07-13
- Bounded Context：**Cloud Compute**（taskCloudService）+ **Event Orchestration**（taskEvents）

## 聚合

### WorkspaceMachinePolicy（根）

- ID: `(CompanyID, WorkspaceID)`
- 属性: `IdleRecycleMinutes`, `PreferIdleReuse`, `EnabledAuthorizationIDs []string`
- 不变量:
  - `IdleRecycleMinutes >= 0`
  - 空 `EnabledAuthorizationIDs` = 不限制
- 仓储: `WorkspaceMachinePolicyRepository`（SQLite）

### CloudServerConfig（既有，扩展）

- 新增 `IdleSince *time.Time`
- 行为: `MarkIdle()`, `ClearIdle()`, `IsIdle()`, `IsBusy()`, `IsStarted()`

## 领域服务

| 服务 | 职责 |
|------|------|
| `IdleNodeSelector` | 在 workspace 内选可复用闲置节点（匹配 auth/platform） |
| `IdleRecycleService` | 找出超时闲置节点并编排 stop |
| `AuthorizationGate` | 校验 start-vm 的 authorization_id 是否在策略白名单 |

## 领域事件（复用）

- 复用 `CLOUD_SERVER_STOPPED`（回收时）
- 不新增公网事件类型（MVP）；内部日志足够

## 仓储接口

```go
type WorkspaceMachinePolicyRepository interface {
  Get(companyID, workspaceID string) (WorkspaceMachinePolicy, error) // 无则 DefaultPolicy
  Upsert(p WorkspaceMachinePolicy) error
  ListWithRecycleEnabled() ([]WorkspaceMachinePolicy, error)
}
```

落点目录（实现可在 `taskCloudService/src/`，薄领域类型同包或 `domain/` 子包按现有风格）。
