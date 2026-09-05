# DDD 笔记：task-detail-machine-owner-hint

## 限界上下文

- **Cloud Runtime**（taskCloudService）：`CloudServerConfig` / `OwnedMachineNode` / `SharedInstanceBinding`

## 概念

| 概念 | 说明 |
|------|------|
| OwnedMachineNode | CSC 上 `instance_id` 非空且未 terminal_released 的绑定 |
| MachineOwnerHint | 查询投影：同实例绑定任务列表 + 当前任务容器是否运行 |
| SharedInstanceBinding | 多 CSC 共享同一 `instance_id`（历史/边缘） |

## 事件

无新增。本需求为只读投影展示（NFR 书面例外）。

## 架构变更影响

无新组件；不更新 ArchiMate / PlantUML。
