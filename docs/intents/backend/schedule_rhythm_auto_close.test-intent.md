# 测试意图：排队调度节奏自动关闭

- 对应意图：`schedule_rhythm_auto_close.intent.md`

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | PATCH 顶层 `auto_close=true` | GET 回显 true；事件 ScheduleRhythmUpdated |
| T2 | 子任务写 auto_close | 拒绝 |
| T3 | 在窗且距结束 4 分钟、有槽位、未 warn | 调用 closing-soon；写 warn_key；事件 Warned |
| T4 | 同窗口周期再次 ticker | 不再 warn |
| T5 | 窗外有槽位、未 release | shutdown 或 stop-vm；清槽；事件 Released |
| T6 | `auto_close=false` 窗外有槽位 | 不释放 |
| T7 | 前端勾选保存 | 请求体含 `auto_close: true`；testid `schedule-rhythm-auto-close` |
