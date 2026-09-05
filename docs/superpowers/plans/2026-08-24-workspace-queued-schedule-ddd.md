# 工作空间级排队调度 — 领域建模（DDD）

- 日期：2026-08-24

## 聚合与实体
- **WorkspaceScheduleRhythm**（新聚合根）：workspace_id / tenant_id / enabled / timezone / windows[]。业务意图：工作空间级调度节奏（此前是顶层任务级 TopTaskScheduleRhythm，保留作 legacy）
- **QueuedAutoRunMembership**（既有聚合，不变）：task_id / workspace_id / top_task_id / depth / status；加入/离开队列意图不变
- **QueuedMachineSlot**（既有）：已占用并发槽位

## 领域事件契约（→ publish 投递，MQ 走既有 publishDomainEvent 通道）
| 事件 | 触发 | 载荷 |
|---|---|---|
| `WorkspaceScheduleSaved`（新） | PUT 保存节奏成功 | workspace_id, tenant_id, enabled, window_count |
| `TaskQueuedForAutoRun`（既有） | 入队 | task_id, tenant_id, workspace_id, top_task_id, depth, status |
| `QueuedAutoRunDeferred`（既有） | 入队即 deferred / 分发置 deferred | task_id, tenant_id, workspace_id, reason |
| `QueuedAutoRunStarted`（既有） | 分发启动成功 | task_id, tenant_id, workspace_id, top_task_id |
| `TaskDequeuedFromAutoRun`（既有） | 退出/手动启动出队 | task_id, tenant_id, workspace_id, reason |
| `ScheduleAutoCloseReleased`（既有，workspace 变体） | 窗口结束自动关闭 | workspace_id, tenant_id, task_ids, release_key, modes |

## 领域规则（不变式）
1. 窗口时间合法（HH:MM）、同节奏窗口互不交叠（复用 validateWindowsNoOverlap）
2. 有效节奏解析：工作空间节奏存在 → 用之；否则回退顶层任务节奏（legacy）
3. 未启用/不在窗口 → 成员 deferred；在窗口 → 依序（depth, enqueued_at）启动，受在窗口窗口并发上限约束
4. 并发槽位去重：task_queued_machine_slots ON DUPLICATE KEY（既有）
5. 手动启动 → 出队（dequeueOnManualStart，既有）

## 纯查询例外（书面）
- 新 GET queue-schedule 快照（节奏+成员+槽位）为读模型聚合，不触发领域事件（既有 handleGetQueuedAutoRun 同款例外）
