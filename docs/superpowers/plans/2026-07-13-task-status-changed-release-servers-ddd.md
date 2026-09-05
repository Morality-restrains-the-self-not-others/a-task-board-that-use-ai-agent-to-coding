# DDD 领域模型：任务状态变更与终态释放

- 日期：2026-07-13
- 设计：`docs/superpowers/specs/2026-07-13-task-status-changed-release-servers-design.md`

## Bounded Contexts

| Context | 职责 |
|---------|------|
| Task | 任务状态 SSOT（progress_column_id / completed） |
| Cloud Runtime | CloudServerConfig、ECS/relay/mock 生命周期 |
| Domain Events | 事件传输与 intent 消费 |

## Aggregates / Entities

- **Task**（taskTaskService）：`id`, `tenant_id`, `workspace_id`, `progress_column_id`, `completed`
- **CloudServerConfig**（既有）：`(company, task_id)` 唯一；`instance_id`, `server_url`, …

## Value Objects

- **TerminalKind**：`completed` | `cancelled` | none
- **TaskStatusSnapshot**：`progress_column_id`, `completed`, optional `column_name`

## Domain Events

| Event | Producer | Consumer |
|-------|----------|----------|
| `TASK_STATUS_CHANGED` | taskTaskService | `task_status_changed/1_release_servers_on_terminal` |
| `CLOUD_SERVER_STOPPED` | 释放编排（复用） | `cloud_server_stopped/1_process_server_stop` |

## Domain Services

- **DetectTaskStatusChange(old, new) → bool**
- **ResolveTerminalKind(columnName, completedFlip) → TerminalKind**
- **ReleaseServersOnTerminal(task, kind)**：编排 ECS / relay / mock 释放（幂等）

## Ports

- `EventPublisher.Publish(eventType, data, key)`
- `CloudServerConfigPort.Load(tenant, workspace, task)`
- `ProgressColumnPort.ResolveName(workspace, columnID)`（可选；可用列名约定 + completed）
- `ContainerStopPort.StopRelay|StopMock(...)`

## 落点

- 领域判定逻辑：`taskEvents/internal/handlers/taskstatuschanged/`（纯函数可单测）
- 发布适配：`taskTaskService/src/events.go`
- 消费入口：`taskEvents/cmd/task_status_changed/1_release_servers_on_terminal/`
