# 自动调度安排 · 排队任务卡可管理 — 价值流

- 日期：2026-08-27

## Related Value Streams

- `top-deliverable-queued-auto-run-schedule`（2026-07-19）：入队/出队/分发领域能力。本增量**消费**该流，不改后端。
- `workspace-queue-schedule`（2026-08-24）：工作空间级节奏页。本增量在其「排队任务」卡上补管理。
- `workspace-queue-schedule-task-title-link`（2026-08-27）：标题链接。本增量保留链接，加操作列。
- `join_queue_requires_workspace_schedule`（2026-08-27）：任务详情入队前校验 enabled。本卡复用同一判定，未启用时引导本页「调度设置」（已在当前页，不跳转）。

## 用户价值

在自动调度安排页直接把任务加入/移出队列，不必先打开任务详情。

## 增量（单一 MVP）

1. 搜索并加入队列（enabled 门闩）
2. 行内离开队列（确认）
3. 刷新真正重新 GET（修 OPT-20260827-041）

无第二期。

## 步骤与测试点

| 步骤 | 测试文件 | 测试点 |
|------|----------|--------|
| 打开排队任务卡 | WorkspaceQueueSchedule.test.js | 卡片可见、成员行存在 |
| 搜索候选 | QueueMembersCard.test.js | GET search 带 workspace_id；已入队不出现 |
| 未启用加入 | QueueMembersCard.test.js | 不发 PATCH；alert 引导调度设置 |
| 已启用加入 | QueueMembersCard.test.js + useWorkspaceQueueSchedule.test.js | PATCH queued_auto_run=true + Idempotency-Key + 刷新 GET |
| 离开队列 | QueueMembersCard.test.js | confirm 后 PATCH false；取消不发 |
| 刷新 | WorkspaceQueueSchedule.test.js | 点击 refresh-members 再发 GET |

## YAML

`conf/value-stream.yaml` 任务协作流新增 `workspace-queue-schedule-members-manage`（active，指向 QueueMembersCard.test.js）。
