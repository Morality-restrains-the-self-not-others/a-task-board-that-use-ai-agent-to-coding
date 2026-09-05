# 意图：加入自动执行队列前须确认工作空间已启用自动调度

## 背景与目标

任务详情「云端开发」页评论区「加入自动执行队列」目前直接 `PATCH queued_auto_run=true`。工作空间未启用自动调度时，任务会入队但长期 `deferred`（文案「自动调度未启用」），用户误以为已开始排队执行。

目标：点击该按钮时**先判断当前工作空间是否已启用自动调度**；未启用则阻断入队并引导去「自动调度安排」。

## 范围与边界

- 范围：`TaskDetailQueuedScheduleToggle` / `useQueuedAutoRunPanel` 的加入队列点击路径。
- 判定：既有 `GET /api/tenant/{tid}/workspace/{wid}/queue-schedule/` 的 `schedule_rhythm.enabled === true`。无节奏行或 `enabled=false` 均视为未启用。
- 服务端：工作空间已配置节奏且 `enabled=false` 时 `queued_auto_run=true` 返回 **409**（见 `workspace_schedule_disabled_enqueue_conflict`）。无节奏行仍允许 legacy deferred 入队。
- 非目标：不把「当前不在时段」当成未启用。

## 约束与风险

- 纯前端门闹仍保留；服务端 409 为第二道门（OPT-20260827-042）。
- 写操作仍须 `createClickGuard` + `Idempotency-Key`（仅 PATCH）。
- 前往设置须整页 `location.assign`，禁止 `router.push` 冒充链接。
- GET 失败须 `data-traceId`，且不得伪造。
- 纯前端门闹，无新服务端业务意图、无新领域事件。未启用时后端另返回 409（见 `workspace_schedule_disabled_enqueue_conflict`）。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ/契约 | 发布点 | 例外理由 |
|---------|--------|---------|--------|----------|
| 入队前读取工作空间调度启用态 | — | — | taskFE GET queue-schedule | 纯查询，无服务端状态变更 |
| 未启用时阻断入队并引导设置 | — | — | taskFE modal + location.assign | 纯前端门闹，未发 PATCH |
| 已启用后加入队列 | TaskQueuedForAutoRun | 既有 PATCH | taskTaskService `enqueueQueuedAutoRun` | 沿用既有入队事件，本增量不新增 |

## 验收标准

1. 未启用：点击「加入自动执行队列」不发 PATCH；弹出确认（前往设置 / 取消）。
2. 确认「前往设置」：浏览器转到 `/tenant/{tid}/queue-schedule/?workspace_id={wid}`。
3. 取消：留在任务详情，仍显示加入按钮。
4. 已启用：先 GET 再 PATCH `queued_auto_run=true`，行为与现网入队一致。
5. GET 失败：错误节点带 `data-traceId`（有则），不 PATCH。

## 实施计划

1. 导出 `isWorkspaceAutoScheduleEnabled` + `fetchWorkspaceAutoScheduleEnabled`。
2. 加入按钮点击先 GET 再决定 PATCH / 弹窗。
3. 单测覆盖启用/未启用/GET 失败；更新既有 join 测例的 GET mock。

## 变更记录

- 2026-08-27：初版（goal-mode：入队按钮先校验工作空间自动调度启用态）
- 2026-08-27：后端 `enabled=false` 时 409 拒绝入队（OPT-20260827-042）
