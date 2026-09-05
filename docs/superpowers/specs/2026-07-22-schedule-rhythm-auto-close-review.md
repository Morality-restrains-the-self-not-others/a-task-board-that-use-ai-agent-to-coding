# Review：排队调度节奏自动关闭

- 日期：2026-07-22
- 计划：`docs/superpowers/plans/2026-07-22-schedule-rhythm-auto-close-plan.md`

## 对照计划

| 项 | 状态 |
|----|------|
| TTS auto_close 列 + PATCH/JSON | ✅ |
| AutoCloser warn/release + 幂等键 | ✅ |
| Cloud lifecycle proxy 前缀 | ✅ |
| Gateway L0 closing-soon | ✅ |
| onlineServiceJS closing-soon | ✅ |
| Vue 勾选 + 单测 | ✅ |
| 意图 / flows | ✅ |

## Log Audit

| 路径 | 日志 |
|------|------|
| warn/release | `tracelog.LogForwardStage` + domain events |
| 容器 | `console.log` closing-soon / shutdown |

## Intent→Event

| 意图 | 事件 | 证据 |
|------|------|------|
| 更新节奏含 auto_close | ScheduleRhythmUpdated | applyScheduleRhythmFromBody |
| 预告 | ScheduleAutoCloseWarned | notifyAutoCloseWarn |
| 释放 | ScheduleAutoCloseReleased | releaseAutoCloseSlots |

## CRG 风险

- graph_status: sparse（CLI；MCP 未挂载）
- 风险摘要：释放集合限定 `queued_machine_slots`；容器失败回退 stop-vm；幂等键防重复

## 结论

无 critical/important 阻断项；可 Ship。
