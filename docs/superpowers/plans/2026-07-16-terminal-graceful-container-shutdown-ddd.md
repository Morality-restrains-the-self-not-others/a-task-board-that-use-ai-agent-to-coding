# DDD 领域摘录：terminal-graceful-container-shutdown

## Bounded Contexts

| Context | 职责 |
|---------|------|
| Task | 进度终态、发布 TASK_STATUS_CHANGED |
| Cloud Runtime | CSC、SoleContainerGate、request-machine-release |
| Container Runtime | onlineServiceJS shutdown 收尾 |
| Domain Events | 编排 notify / await / hard-release |

## Aggregates / Entities

- **CloudServerConfig**（已有）：instance_id, server_url, terminal_released
- **Task**（已有）：progress_column_id, completed

## Domain Services

- `NotifyContainerTerminalShutdown`
- `RequestMachineRelease`（SoleContainerGate）
- `AwaitGracefulShutdownOrHardRelease`

## Domain Events

| 事件 | 说明 |
|------|------|
| TASK_STATUS_CHANGED | 触发 |
| TASK_GRACEFUL_SHUTDOWN_AWAIT | 轮询/补偿（类 CONTAINER_MIGRATE_AWAIT_READY） |
| CLOUD_SERVER_STOPPED | 节点销毁 |

## Repository

- cloudconfig.LoadForTask / ClearAfterStop（已有）
- listInstanceBindings（已有）
