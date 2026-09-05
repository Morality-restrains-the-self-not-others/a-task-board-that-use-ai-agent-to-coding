# 意图：排队调度节奏自动关闭

- 日期：2026-07-22
- 设计：`docs/superpowers/specs/2026-07-22-schedule-rhythm-auto-close-design.md`
- value-stream step：`schedule-rhythm-auto-close`

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|----------|------------------|--------|---------------|----------|
| 更新顶层调度节奏（含 auto_close） | ScheduleRhythmUpdated | taskTaskService `applyScheduleRhythmFromBody` | 前端节奏面板刷新 | 证据豁免：首期 publishDomainEvent 日志/审计 |
| 窗口结束前预告容器 | ScheduleAutoCloseWarned | taskTaskService `runAutoCloseForTop` | 容器 closing-soon | 证据豁免：首期 publishDomainEvent 日志/审计 |
| 窗口结束自动释放机器 | ScheduleAutoCloseReleased | taskTaskService `runAutoCloseForTop` | 清槽位 / stop-vm 回退 | 证据豁免：首期 publishDomainEvent 日志/审计 |

## 验收要点

1. 仅顶层可写 `schedule_rhythm.auto_close`
2. 勾选后距结束 ≤5 分钟通知槽位容器一次
3. 窗外对槽位任务执行释放一次并清槽
4. 未勾选不自动释放
