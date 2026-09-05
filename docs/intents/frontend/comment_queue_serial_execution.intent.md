# 意图：任务详情自动执行队列与评论逐条执行同卡

## 背景

任务详情将「加入自动执行队列」放在可折叠的任务辅助信息，将「执行依赖」放在添加评论区。两块都表达顺序执行，但用户无法在启用自动调度/入队时理解评论如何一条一条跑。

## 目标

1. 任务详情「添加评论」的 `CommentExecutionDependencyPicker` 内嵌队列 toggle（加入/离开、状态 chip、前往自动调度安排）。
2. `queued_auto_run === true` 时强制新评论草稿 `wait_previous`，禁用「不等待前序」，并展示逐条执行说明。
3. 任务辅助信息不再渲染 `task-queued-auto-run-toggle`。
4. 看板卡片 `showDependencyPicker=false` 不出现队列。
5. 入队/出队成功后 `task-updated` 回填 `localTask`（既有 PATCH，无新 API）。
6. 点击「加入自动执行队列」须先确认工作空间已启用自动调度（见 `join_queue_requires_workspace_schedule.intent.md`）。

## 非目标

- 不改 taskTaskService 分发与 comment_container_bindings 调度
- 不批量改写历史评论 execution_mode

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ/契约 | 发布点 | 例外理由 |
|---------|--------|---------|--------|----------|
| 加入/离开自动执行队列 | 既有 queued_auto_run PATCH | — | taskTaskService | 本增量仅搬 UI；领域副作用沿用既有 PATCH |
| 评论依赖草稿 | CommentExecutionModeChanged | 发评时 PATCH | TTS / taskAIComment | 入队只约束草稿，发评仍走既有路径 |

## 变更记录

- 2026-08-27：初版（goal-mode：队列与执行依赖同卡 + 入队锁定串行）
- 2026-08-27：入队按钮先校验工作空间「启用自动调度」；未启用阻断 PATCH 并引导设置页
