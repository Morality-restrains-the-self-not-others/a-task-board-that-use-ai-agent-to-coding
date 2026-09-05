# 测试意图：创建任务自动运行时可选择加入自动调度队列

对应：`docs/intents/frontend/create_task_queued_auto_run.intent.md`

## 测试目标

验证创建/保存任务在「自动运行 + 工作空间已启用自动调度」时展示入队选项；勾选后入队且不立即启服；未勾选保持立即启服。

## 测试分层

| 层 | 覆盖 |
|----|------|
| 单元（JS） | 启用判定、选项可见性、payload、hint；插件 `workspace-auto-schedule` / `create-task-payload` |
| 组件（Vue） | 嵌套勾选出现/隐藏、默认未勾、关闭 auto_run 清队列勾选 |
| 插件 | 浮窗 markup + GET `getQueueSchedule` + payload `queued_auto_run` |
| Go | 创建 `queued_auto_run=true` 入队且无 start-vm；仅 auto_run 仍启服；force 仍启服 |

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | `schedule_rhythm.enabled=true`，auto_run 可勾选且已勾 | 渲染创建模态 | 出现 `task-queued-auto-run-checkbox`，未勾选 |
| T2 | enabled=false 或无节奏 | 渲染 | 不出现队列勾选 |
| T3 | 队列勾选可见且用户勾选 | 提交 | payload `queued_auto_run===true` |
| T4 | auto_run true 未勾队列 | 提交 | 不发送 true（false 或不入队） |
| T5 | auto_run 改为 false | — | `queued_auto_run` 置 false，勾选隐藏 |
| T6 | GET 失败带 X-Trace-Id | 打开模态 | 不展示勾选；错误节点 `data-traceId` |
| T7 | POST auto_run+queued_auto_run | 创建 | 201、membership 存在、无 start-vm |
| T8 | POST 仅 auto_run | 创建 | 仍触发 start-vm（回归） |
| T9 | 入队事件 | 创建 queued | `TaskQueuedForAutoRun`（既有 enqueue） |
| T10 | 插件 auto_run + schedule enabled | 勾选队列后创建 | payload `queued_auto_run===true` |

## 数据与环境

- 前端 vitest + jsdom；Go `setupTestDB` + `startAutoRunMockServices`。
- GET mock：`/queue-schedule/` 返回 `{ schedule_rhythm: { enabled } }`。

## 通过标准

上表 T1–T9 全绿；既有 `TestCreateTaskAutoRunTriggersStart` 仍通过。
