# 测试意图：未启用工作空间自动调度时拒绝 queued_auto_run 入队

对应：`docs/intents/frontend/workspace_schedule_disabled_enqueue_conflict.intent.md`

## 测试目标

验证工作空间已配置自动调度但未启用时，入队失败（409）且不写 membership；已启用或无节奏行（legacy）行为不变。

## 测试分层

| 层 | 覆盖 |
|----|------|
| Go 单元 | `enqueueQueuedAutoRun` 拒绝/允许/legacy |
| Go HTTP | POST 创建 `queued_auto_run=true` → 409 |

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | workspace rhythm `enabled=false` | `enqueueQueuedAutoRun` | 错误 HTTP 409，无 membership |
| T2 | `enabled=true` | `enqueueQueuedAutoRun` | membership 存在 |
| T3 | 无 workspace rhythm | `enqueueQueuedAutoRun` | 成功，status=`deferred` |
| T4 | POST create `queued_auto_run=true` + disabled | `handleTaskRoutes` | 409，body 含「尚未启用自动调度」，membership 行数为 0 |

## 数据与环境

- `setupTestDB` + `setupWorkspaceRhythmAllDay`；HTTP 测例用 `startAutoRunMockServices`。

## 通过标准

上表 T1–T4 全绿；`TestCreateTaskAutoRunQueuedDefersStart`（enabled=true）仍 201。
