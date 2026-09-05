# 意图：自动调度安排页展示调度历史

## 背景与目标

`/tenant/:tenant/queue-schedule/` 状态栏（`data-testid="schedule-status-bar"`）只展示当前是否在允许运行时段与槽位占用。用户希望看到该工作空间近期调度轨迹（入队、改时段、进入/离开窗口、自动启服）。

## 范围与边界

- 范围内：状态栏下方历史卡；GET 快照 `recent_history`；GET `.../queue-schedule/history/` 分页；只读。
- 范围外：不改节奏表单、排队任务卡管理、出队算法、启服逻辑本身。

## 约束与风险

- 禁止后台轮询（元规则 51）；刷新沿用页面「刷新」。
- 任务标题用真实 `a[href]`。
- 错误节点 `data-traceId`。
- 加载更多：L1 锁 + `Anti-Replay-OK: read-pagination`。

## 验收标准

1. 有历史时卡片列出时间、类型、文案；任务行可点进详情。
2. 无历史时空态可见，状态栏仍展示当前时段/槽位。
3. 「加载更多」发 GET history 且带 cursor；无更多则按钮消失。
4. 失败时错误节点带 `data-traceId`（有则）。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ/契约 | 发布点 | 例外 |
|---------|--------|---------|--------|------|
| 打开页面看历史 | — | — | GET 快照/history | 纯查询 |
| 加载更多 | — | — | GET history | 纯查询 |

写路径事件见 `docs/intents/backend/queue_schedule_history.intent.md`。

## 变更记录

- 2026-08-28：新增。
