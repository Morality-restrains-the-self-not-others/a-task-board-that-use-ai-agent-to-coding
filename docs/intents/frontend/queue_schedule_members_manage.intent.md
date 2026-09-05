# 意图：自动调度安排排队任务卡可加入/离开队列

## 背景与目标

`/tenant/:tenant/queue-schedule/`「排队任务」卡（`data-testid="queue-members-card"`）只展示成员，不能管理自动调度。用户期望在此搜索加入队列、将任务移出队列。

## 范围与边界

- 范围内：排队任务卡加入/离开、工作空间内任务搜索、未启用阻断入队、刷新真正 GET。
- 范围外：不改节奏表单；不新增后端 API；不改出队顺序算法；不在此启停云主机。

## 约束与风险

- 写操作：`createClickGuard` + `Idempotency-Key`。
- 离开须 `modalService.confirm`，禁止 `window.confirm`。
- 导航仍用真实 `a[href]`（既有标题链接）。
- 搜索由用户输入触发（debounce），禁止 setInterval 轮询。
- 错误展示 `data-traceId`。

## 验收标准

1. 卡内可搜索本工作空间任务并「加入队列」；已在队中的不出现。
2. 工作空间未启用自动调度时加入不发 PATCH，提示先保存「调度设置」。
3. 行内「离开队列」确认后 PATCH `queued_auto_run=false` 并刷新列表；取消不发。
4. 写请求带 `Idempotency-Key`；失败节点有 `data-traceId`（有则）。
5. 「刷新」发出 GET `/queue-schedule/`。

## 实施计划

见 `docs/superpowers/plans/2026-08-27-queue-schedule-members-manage-plan.md`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 搜索候选任务 | — | — | 前端 GET search | — | 纯查询 |
| 未启用阻断入队 | — | — | 前端 alert | — | 未发 PATCH |
| 从排队卡加入队列 | TaskQueuedForAutoRun | 既有 PATCH | taskTaskService `enqueueQueuedAutoRun` | dispatcher 纳入候选 | 沿用既有事件，本增量不新增 |
| 从排队卡离开队列 | TaskDequeuedFromAutoRun | 既有 PATCH | taskTaskService `dequeueQueuedAutoRun` | 出队 | 同上 |
| 刷新队列快照 | — | — | 前端 GET queue-schedule | — | 纯查询 |

## 变更记录

- 2026-08-27：新增。排队任务卡支持加入/离开自动调度队列。
