# 自动调度安排 · 排队任务卡可管理 — 设计文档

- 日期：2026-08-27
- 入口：/goal（跳过 USER GATE）
- 范围：taskFE 前端（复用既有 taskTaskService API，无新后端）
- 页面：`/tenant/:tenant/queue-schedule/`（壳标题「云端开发」）
- 元素：`data-testid="queue-members-card"`（「排队任务」白卡片）

## 架构基线理解

当前架构 current 为 **v114**（企业景观 / 应用集成）。本增量**不改变**服务边界、数据所有权或事件通道：继续由 taskFE → APISIX → taskTaskService 读写 `task_queued_auto_run_memberships`。不新增 ArchiMate 版本。

CRG：`code-review-graph update --brief` 成功（8 files / 0 nodes）。无 `codegraph_explore` MCP；基线来自源码。

## 问题

排队任务卡目前只展示成员表 +「刷新」。加入/离开队列只存在于任务详情 `TaskDetailQueuedScheduleToggle`。用户在「自动调度安排」页看着队列，无法在此管理（入队/出队）。刷新按钮还绑定了未 `return` 的 `loadSnapshot`（OPT-20260827-041），点击无请求。

## 方案（采用）

在「排队任务」卡内完成队列成员管理，**不新增 API**：

| 能力 | 交互 | 既有契约 |
|---|---|---|
| 加入队列 | 「加入队列」展开搜索 → 选任务 → PATCH | `PATCH /api/tenant/{tid}/workspace/{wid}/todos/{taskId}/` body `{ queued_auto_run: true }` |
| 离开队列 | 行内「离开队列」→ `modalService.confirm` → PATCH | 同上 `queued_auto_run: false` |
| 搜索候选 | 用户输入触发（debounce 250ms，非轮询） | `GET /api/tasks/search/tenant_id/{tid}/?q=&workspace_id=&limit=20` |
| 启用门闩 | 入队前读当前页已加载 snapshot | `schedule_rhythm.enabled === true`（复用 `isWorkspaceAutoScheduleEnabled`）；未启用则 alert 引导上方「调度设置」，不发 PATCH |
| 刷新 | 调用 `reload()`（修 OPT-041） | GET queue-schedule |

节奏时段/并发仍在上方「调度设置」卡，本卡不重复表单。

## 拒绝的方案

1. **新 workspace 级 join/leave API** — 已有 PATCH todos，重复契约无收益。
2. **把调度设置并入本卡** — 用户点的是排队任务表；设置卡已存在。
3. **拖拽重排** — 出队顺序由后端 `depth ASC, enqueued_at ASC` 决定；改顺序需新字段，超出本增量。

## 前端结构

- 抽出 `QueueMembersCard.vue`（页面已近 500 行门禁）。
- `useWorkspaceQueueSchedule` 增加 `patchQueuedAutoRun(taskId, queued, idempotencyKey)`，成功后 `loadSnapshot`。
- 写按钮：`createClickGuard` + `Idempotency-Key`；刷新：L1 锁 + `Anti-Replay-OK: read-refresh`。
- 离开确认：`modalService.confirm`（禁止 `window.confirm`）。
- 错误节点 `data-traceId`。
- 搜索结果过滤：同 `workspace_id`、排除已在 `members` 的 `task_id`。

## 事件

入队/出队沿用既有 `TaskQueuedForAutoRun` / `TaskDequeuedFromAutoRun`（taskTaskService PATCH 路径）。纯前端搜索/门闩无新事件。

## 测试

- `QueueMembersCard.test.js`：搜索入队、未启用不 PATCH、离开确认、Idempotency-Key、错误 data-traceId。
- `useWorkspaceQueueSchedule.test.js`：PATCH + 刷新快照。
- `WorkspaceQueueSchedule.test.js`：刷新走 GET；卡片仍渲染。

## 架构变更

无。不新增服务/表/路由/Kafka topic。
