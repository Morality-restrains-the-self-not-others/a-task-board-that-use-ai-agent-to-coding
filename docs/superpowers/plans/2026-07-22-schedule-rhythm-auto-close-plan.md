# 实施计划：排队调度自动关闭

## 事件契约

| 事件 | 关键字段 |
|------|----------|
| ScheduleRhythmUpdated | + auto_close |
| ScheduleAutoCloseWarned | top_task_id, task_ids[], warn_key |
| ScheduleAutoCloseReleased | top_task_id, task_ids[], release_key, mode |

## 爆炸半径与测试缺口

- 文件：`queued_schedule*.go`、`cloud_client.go`、`container_gateway_proxy.go`、`l0_registry.go`、`TaskDetailQueuedSchedulePanel.vue`、`taskLifecycleShutdown.mjs` / routes
- 缺口：无 auto_close 单测 / 前端勾选测 → 本计划补齐

## 任务

- [x] TTS：列 `auto_close`/`*_key`；JSON/PATCH
- [x] TTS：`minutesUntilWindowEnd` + `runAutoCloseForTop` + ticker 挂钩
- [x] TTS：`stopVM` / `notifyContainerClosingSoon` / `shutdownContainer` 客户端（可测替身）
- [x] TTS：单测 T1–T6
- [x] Cloud：proxy 前缀 `container-task-lifecycle-`
- [x] Gateway：L0 closing-soon
- [x] onlineServiceJS：`/task-lifecycle/closing-soon`
- [x] Vue：勾选 + 单测
- [x] 意图/flows 更新
- [x] Review + PR
