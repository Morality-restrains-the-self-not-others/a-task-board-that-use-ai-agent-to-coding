# 自动调度安排 · 调度历史 — 实施计划

- 日期：2026-08-28
- 设计：`docs/superpowers/specs/2026-08-28-queue-schedule-history-design.md`

## 任务

- [ ] **T1 意图/INDEX/value-stream/架构 v115**  
  意图已写；补 INDEX、value-stream.yaml、v115 四件套、VERSION_HISTORY。

- [ ] **T2 Red：history store**  
  `queued_schedule_history_test.go`：append + list 分页 + 跨 workspace 隔离。

- [ ] **T3 Green：DDL + store**  
  `dataMigrate/taskTaskService/017_queued_schedule_history.sql`；`queued_schedule_history.go`。

- [ ] **T4 Red/Green：写路径**  
  enqueue/dequeue/save/start/auto-close/window flip 各断言一行；空扫不写。

- [ ] **T5 Red/Green：HTTP**  
  GET history 200/403；快照 `recent_history`；openapi。

- [ ] **T6 Red：前端卡**  
  ScheduleHistoryCard + composable loadMore。

- [ ] **T7 Green：接入页面**  
  状态栏下渲染卡片；页面测例。

- [ ] **T8 验证**  
  Go 测 + vitest；gofmt；登记精准重启 task-task-service + taskFE。
