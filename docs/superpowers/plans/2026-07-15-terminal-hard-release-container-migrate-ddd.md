# DDD 领域建模摘录：终态硬释放 + 容器迁移

## Bounded Contexts

| BC | 职责 |
|----|------|
| Task | 进度终态判定与 `TASK_STATUS_CHANGED` |
| Cloud Runtime | CSC、reuse、bindings、migrate、terminal_released、stop |
| Domain Events | 编排 migrate→release |

## Aggregates / Entities

- **CloudServerConfig**（CSC）：`instance_id`, `server_url`, `terminal_released`
- **WorkspaceMachinePolicy**：`prefer_idle_reuse`（不变）

## Domain Services

- `ForeignContainerMigrate` — 将 busy 兄弟迁到 OwnedMachineNode
- `TerminalHardRelease` — mark + stop；禁止 reuse
- `IdleReuseBind` — 绑定目标并 **UnbindSource**

## Value Objects

- `TerminalKind` = completed | cancelled
- `InstanceID`
- `ForbidReuseInstanceID`

## Repository 接口（逻辑）

- `ListBindingsByInstance(company, workspace, instanceID)`
- `MarkTerminalReleased(company, workspace, taskID)`
- `MigrateContainerOffInstance(...)`

## 不变式

1. `terminal_released=1` ⇒ 不可被 idle reuse 选中
2. 硬释放前：同 instance 无其他 busy `server_url`
3. migrate 目标 instance ≠ `forbid_reuse_instance_id`
