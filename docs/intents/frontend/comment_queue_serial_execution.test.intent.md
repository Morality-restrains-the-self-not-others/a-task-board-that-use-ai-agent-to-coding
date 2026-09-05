# 测试意图：任务详情自动执行队列与评论逐条执行同卡

## 用例

### T1 — 未入队：同卡展示队列与依赖

- **给定** 任务详情 composer、`queued_auto_run=false`、有 tenant/workspace/task id
- **当** 渲染 `CommentExecutionDependencyPicker`
- **则** 可见 `task-queued-auto-run-toggle` 与「加入自动执行队列」；「不等待前序」可选；可见未入队逐条说明 `comment-dep-queue-serial-hint`

### T2 — 已入队：锁定等待前序

- **给定** `queued_auto_run=true`
- **当** 渲染 picker
- **则** 「不等待前序」disabled；`executionMode` 为 `wait_previous`；hint 含「一条一条执行」；可点「离开队列」

### T3 — 入队后从 independent 切回 wait

- **给定** 草稿为 independent，随后 `task.queued_auto_run` 变为 true
- **当** 响应式更新
- **则** `executionMode` 变为 `wait_previous`，depends 清空策略与切 wait 一致

### T4 — 辅助信息不再含队列

- **给定** 任务详情辅助信息面板
- **当** 展开正文
- **则** 面板内无 `task-queued-auto-run-toggle`；队列仅在评论 composer 的 picker 内

### T5 — 看板卡片不出现队列

- **给定** `TaskCardCommentsSection` `showDependencyPicker=false`
- **当** 渲染 composer
- **则** 无 dependency picker、无队列 toggle
