# 加入自动执行队列须先确认工作空间自动调度已启用

- 日期：2026-08-27
- 入口：/goal（跳过 USER GATE）
- 范围：taskFE 任务详情「加入自动执行队列」按钮
- 架构变更：无（不新增服务/表/端点；不更新 ArchiMate）

## Context

按钮 `data-testid="queued-auto-run-join"` 当前直接 PATCH 入队。工作空间「自动调度安排」未勾选「启用自动调度」时，调度器只会把成员标成 deferred。用户期望：**先判断当前工作空间是否已启动自动调度**。

## Decision

1. **门闹时机**：点击加入时同步 GET 既有 `GET /api/tenant/{tid}/workspace/{wid}/queue-schedule/`（用户操作触发，非轮询）。
2. **判定**：`schedule_rhythm?.enabled === true` 才允许 PATCH。无节奏或 `enabled=false` → 未启用。当前不在时段**仍允许入队**（等待窗口）。
3. **未启用 UX**：`modalService.confirm`（禁止原生 confirm）。确认「前往设置」→ `window.location.assign(schedulePageRoute)`（真实整页跳转）。取消 → 不 PATCH。
4. **已启用**：沿用既有 `PATCH queued_auto_run=true` + clickGuard Idempotency-Key。
5. **GET 失败**：inline `queued-auto-run-save-error` + `data-traceId`，不 PATCH。
6. **后端**：本增量不改 `enqueueQueuedAutoRun`（其它客户端仍可入队为 deferred）。

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 页面加载时预取 enabled 并禁用按钮 | 额外读请求且状态易陈旧；用户要求的是点击时判断 |
| 后端 409 拒绝未启用入队 | 改变既有 API 与「先入队再开调度」运维路径；本增量只改按钮 |
| 用任务 JSON 的 legacy `schedule_rhythm` | 节奏已迁工作空间级，任务字段不能代表工作空间启用态 |

## Consequences

- 正面：未开调度的工作空间不会再被「假入队」误导。
- 负面：加入前多一次 GET；网络失败时用户需重试。
- 权限：GET 已有 `hasWorkspaceAccess`，与 PATCH 同一工作空间成员边界。
