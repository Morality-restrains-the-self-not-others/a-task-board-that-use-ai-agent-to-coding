# DDD：Chrome 插件项目单选 + 自动运行控件

- **Date:** 2026-08-27
- **Bounded context:** taskChromePlugin（前端扩展，非后端 BC）

## 模型

| 类型 | 名称 | 说明 |
|------|------|------|
| 既有 VO | `ProjectAutoRunCapability` | `projectAllowsAutoRun(project)` |
| 新增 VO | `AutoRunControlState` | `{ enabled, checked, hint }` |
| 领域服务（纯函数） | `resolveAutoRunControlState` | 项目 + 勾选偏好 → 控件状态 |
| 领域服务（纯函数） | `pickSingleProjectId` | 多候选收缩为至多一个有效 id |

## 事件

| 业务意图 | 领域事件 | MQ | 例外 |
|----------|----------|-----|------|
| 单选项目并钳制自动运行 | （无） | — | **书面例外**：纯客户端 UI，成功路径不投递 MQ；创建任务仍走既有 TASK 创建事件 |

## 端口

无新仓储/适配器。创建任务出站端口不变。
