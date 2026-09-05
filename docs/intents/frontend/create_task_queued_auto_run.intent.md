# 意图：创建任务自动运行时可选择加入自动调度队列

- 日期：2026-08-27
- 入口：/goal（跳过 USER GATE）
- 页面：`/tenant/:tenant/work-panel/` 创建任务模态（「是否自动运行」）

## 背景与目标

创建任务勾选「是否自动运行」后会立即按项目运行模版启服。工作空间若已启用「自动调度」，任务详情才能「加入自动执行队列」，创建时看不到该选择，容易立刻起机、错过排队时段。

目标：创建/保存任务时，若 **自动运行已启用** 且 **工作空间自动调度已启用**，展示「加入自动调度队列」选项，由用户决定是否入队。

## 范围与边界

- 范围：`CreateTaskAutoRunSection` / `useCreateTaskAutoRun` / 创建提交 payload；Chrome 插件浮窗与 DevTools 单请求/批量创建；taskTaskService 创建/更新在 `queued_auto_run=true` 时跳过立即 start-vm。
- 判定：`GET .../queue-schedule/` 的 `schedule_rhythm.enabled === true`（与 `join_queue_requires_workspace_schedule` 一致）。无节奏行或 `enabled=false` 不展示选项。
- 默认：不勾选（保持现网立即启服）。勾选后 POST/PUT `queued_auto_run=true`。
- 非目标：不把「当前不在时段」当成未启用。

## 约束与风险

- 入队 + 立即 auto_run 会双启服：勾选队列时服务端 **跳过立即 start-vm**，由 queued dispatcher 按时段启动。`force_auto_run` 仍立即启服。
- GET 失败：隐藏选项、错误节点带 `data-traceId`（有则），不阻断创建。
- 无新 API；入队沿用既有 `applyQueuedAutoRunFromBody` → `TaskQueuedForAutoRun`。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ/契约 | 发布点 | 例外理由 |
|---------|--------|---------|--------|----------|
| 读取工作空间调度启用态 | — | — | taskFE GET queue-schedule | 纯查询 |
| 创建时加入自动调度队列 | TaskQueuedForAutoRun | 既有 publishDomainEvent | taskTaskService `enqueueQueuedAutoRun` | 证据豁免：沿用既有入队事件，本增量不新增 Kafka 契约 |
| 入队后跳过立即启服 | — | 结构化日志 `task_auto_run_deferred_to_queue` | create/update 跳过 `scheduleTaskAutoRunFn` | 无新领域状态；启服推迟到既有 `QueuedAutoRunStarted` |

## 验收标准

1. 工作空间未启用自动调度：即使勾选自动运行，也不出现「加入自动调度队列」。
2. 已启用且勾选自动运行：出现嵌套勾选，默认未勾；hint 说明入队后不立即启服。
3. 勾选后提交：body 含 `queued_auto_run: true` 且 `auto_run: true`；任务入队；不调用立即 start-vm。
4. 不勾选队列：行为与现网一致（立即 auto_run 启服）。
5. 关闭自动运行：队列勾选隐藏且 `queued_auto_run` 置 false。
6. GET 失败：不展示选项；错误带 `data-traceId`（有则）。

## 实施计划

1. 纯函数：`isWorkspaceAutoScheduleEnabled`、`shouldShowCreateTaskQueuedAutoRun`、`appendQueuedAutoRunToCreatePayload`。
2. 模态打开时 GET queue-schedule（不轮询）。
3. UI 嵌套勾选；提交带 `queued_auto_run`。
4. 服务端 `deferImmediateAutoRunStart`：queued 且非 force 则跳过 start-vm。

## 变更记录

- 2026-08-27：初版（goal-mode：创建任务自动运行 × 工作空间自动调度 → 可选入队）
- 2026-08-27：Chrome 插件浮窗/DevTools 同步「加入自动调度队列」（OPT-20260827-043）
