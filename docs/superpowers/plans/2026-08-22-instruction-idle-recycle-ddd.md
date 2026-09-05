# DDD：指令闲置回收

- **日期:** 2026-08-22
- **Bounded Contexts:** Cloud（机器生命周期）、Credential（容器引导契约）、Container Runtime（onlineServiceJS）
- **落点:** 扩展现有 `taskCloudService/src` 与 Credential application，不新建微服务

## 聚合

### WorkspaceMachinePolicy（已有）

- ID: `(CompanyID, WorkspaceID)`
- `IdleRecycleMinutes`：0=关闭指令闲置与卸载闲置扫描

### Comment-scoped CSC（已有，扩展）

- ID: CSC `id`（评论级）
- 新增 `InstructionIdleSince *time.Time`
- 不变量：
  - 仅交付成功可 Mark
  - `minutes==0` 禁止 Mark
  - 新指令 / busy heartbeat false 必须 Clear
  - 释放走 `releaseMachineForTerminal`（Stopped 幂等）

### InstructionIdleWindow（值对象）

- `Enabled bool`, `Minutes int`（minutes>0）
- 由 Policy 投影到 task-detail

### MachineReleaseSTS（值对象，可选）

- 短时 AK/SK/Token + Expiration + InstanceID
- 仅 CPA `StsReleaseRoleArn` 非空时存在
- 不属于评论聚合

## 领域服务

| 服务 | 职责 |
|------|------|
| `InstructionIdleService` | Mark/Clear 列 + 发 Marked/Cleared |
| `InstructionIdleRecycleService` | 扫描到期 CSC，调用既有 release |
| `WorkspacePolicyReader` | Credential 端口：按租户+工作空间读分钟数 |

## 端口

```go
type WorkspaceMachinePolicyFetcher interface {
  FetchIdleRecycleMinutes(companyID, workspaceID string) (int, error)
}

type InstructionIdleStore interface {
  Mark(cfgID string, at time.Time) error
  Clear(cfgID string) error
}
```

云 SDK / Kafka 仅适配器：`publishDomainEvent`、`releaseMachineForTerminal`。

## 领域事件

| 事件 | 载荷键（幂等） | 消费者 |
|------|----------------|--------|
| CONTAINER_INSTRUCTION_IDLE_MARKED | config_id | 审计（可无 handler） |
| CONTAINER_INSTRUCTION_IDLE_CLEARED | config_id | 审计 |
| CLOUD_SERVER_STOPPED | stop_request_id | 现有 process-server-stop |

## 应用服务编排

1. Credential `FetchTaskDetail` → PolicyFetcher → DTO 字段
2. Cloud heartbeat → InstructionIdleService
3. taskEvents timer → recycleIdleMachines + recycleInstructionIdleMachines
4. OSJS：交付门闩 → heartbeat idle → setTimeout → L1/L3
