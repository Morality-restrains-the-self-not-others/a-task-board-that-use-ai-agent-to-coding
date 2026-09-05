# 意图：未启用工作空间自动调度时拒绝 queued_auto_run 入队

- 日期：2026-08-27
- 入口：OPT-20260827-042
- 页面/API：`POST/PATCH .../todos/` 的 `queued_auto_run=true`；`enqueueQueuedAutoRun`

## 背景与目标

任务详情「加入自动执行队列」已在前端 GET `queue-schedule` 且仅 `schedule_rhythm.enabled===true` 时 PATCH。`enqueueQueuedAutoRun` 仍允许工作空间已配置节奏但 `enabled=0` 时写入 membership 并标 `deferred`，其它客户端或旧 SPA 可绕过按钮门闹。

目标：工作空间**已配置节奏且未启用**时，入队返回 **409**，不插入 membership。文案与前端弹窗对齐。

## 范围与边界

- 范围：`enqueueQueuedAutoRun`；创建/更新经 `applyQueuedAutoRunFromBody` 透传 HTTP 状态。
- 判定：`loadEffectiveWorkspaceRhythm` 得到 `wsScoped && !Enabled`。
- 非 409：无工作空间节奏行（legacy 回退任务级，仍可 deferred 入队）；已启用但不在时段（仍入队 `deferred`）。
- 非目标：改 queue-schedule GET；把「当前不在时段」当成未启用。

## 约束与风险

- 创建路径入队失败须删除已插入的任务行（与 `saveTaskProjects` 失败路径一致），避免 409 后留下孤儿任务。
- 文案 SSOT：`errMsgWorkspaceAutoScheduleDisabled`。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ/契约 | 发布点 | 例外理由 |
|---------|--------|---------|--------|----------|
| 未启用时拒绝入队 | — | — | `enqueueQueuedAutoRun` 返回 409 | 无状态变更，不发 `TaskQueuedForAutoRun` |
| 已启用后入队 | TaskQueuedForAutoRun | 既有 publishDomainEvent | `enqueueQueuedAutoRun` | 证据豁免：沿用既有入队事件 |

## 验收标准

1. 工作空间节奏存在且 `enabled=false`：`enqueueQueuedAutoRun` 报错，无 membership。
2. 同上，POST `queued_auto_run=true`：HTTP 409，响应含「尚未启用自动调度」，无 membership。
3. `enabled=true`：入队成功。
4. 无工作空间节奏：仍可 legacy deferred 入队（不 409）。

## 实施计划

1. `httpStatusError` + `errWorkspaceAutoScheduleDisabled`（409）。
2. `enqueueQueuedAutoRun` 在 `wsScoped && !Enabled` 时返回该错误。
3. create/update 用 `httpStatusOf`；创建失败回滚任务行。
4. Go 单测覆盖拒绝/允许/legacy/HTTP 409。

## 变更记录

- 2026-08-27：初版（OPT-20260827-042）
