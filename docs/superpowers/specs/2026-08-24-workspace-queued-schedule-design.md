# 工作空间级排队调度 — 设计文档

- 日期：2026-08-24
- 入口：/goal（跳过 USER GATE）
- 范围：taskFE（前端）、taskTaskService（后端）、dataMigrate（迁移 SQL）
- 背景：生产环境「排队调度设置」位于任务详情（TaskDetailQueuedSchedulePanel，data-testid=`task-queued-schedule-open`）。用户要求：
  1. 将该功能从任务详情**拆出**，放到**工作空间左侧**（租户控制台侧边栏 Sidebar），标题「工作空间的排队调度」
  2. 调度语义改为**工作空间级**：工作空间启用自动调度后，在设置的时间窗内**按顺序逐个启动**已加入调度的任务；未启用/不在窗口 → deferred
  3. 任务加入调度的机制保留（PATCH queued_auto_run）

## 现状（代码基线）

### 数据模型（taskTaskService）
- `task_top_deliverable_schedule_rhythms(task_id PK, tenant_id, workspace_id, enabled, timezone, updated_at)` — 节奏按**顶层任务**维度
- `task_top_deliverable_schedule_rhythm_windows(id PK, task_id, daily_start, daily_end, max_queued_machines, auto_close, ...)` — 时间窗按任务维度
- `task_queued_auto_run_memberships(task_id PK, tenant_id, workspace_id, top_task_id, depth, status, enqueued_at)` — 队列成员（任务级，top_task 分组）
- `task_queued_machine_slots(task_id, top_task_id, acquired_at)` — 已占用并发槽位

### 后端（taskTaskService/src/）
- `queued_schedule.go`：scheduleRhythm/Window 结构、loadScheduleRhythm/Windows、upsert、`applyScheduleRhythmFromBody`（PATCH task body `schedule_rhythm`）、`rhythmInWindow`、`enqueueQueuedAutoRun`（初始 status 由 top-task 节奏决定）、`deferredReason`、`countQueuedSlots`、`listMembershipsForTop`、taskJSON 的 queued 字段注入
- `queued_schedule_dispatch.go`：`runQueuedScheduleDispatchOnce`（按 top_task 分组 → `dispatchTopQueue`：节奏未启用/不在窗口 → 全体 deferred；在窗口 → 依序启动至 maxN）、`handleInternalQueuedScheduleDispatchOnce`（内部端点 `/api/internal/tasks/queued-schedule/dispatch-once/`，taskEvents timer 触发）
- `queued_schedule_auto_close.go`：`runQueuedScheduleAutoCloseOnce`（按 task 级节奏 auto_close 窗口扫机器停止）、`runAutoCloseForTop`
- `route_handlers.go`：GET `/todos/{taskId}/queued-auto-run/` → `handleGetQueuedAutoRun`（节奏+成员+槽位快照）
- `main.go`：`/api/tenant/{tid}/workspace/{wid}/todos...` → handleTaskRoutes；`/api/tenant/{tid}/workspace/{wid}/queue-schedule` 尚不存在

### 前端（taskFE/app/src/）
- `components/task-detail/TaskDetailQueuedSchedulePanel.vue`（499 行）：按钮「排队调度设置」+ 模态（节奏窗口设置 + 加入/离开队列 + 队列列表）。嵌入于 `TaskDetailTaskAuxInfoPanel.vue:265`
- `components/task-detail/useQueuedAutoRunPanel.js`：composable（入队/出队/快照加载，GET `/todos/{topTaskId}/queued-auto-run/`）
- `views/WorkPanel.vue` + `WorkPanelHeader.vue`：工作面板（无自有左侧栏）
- `components/Sidebar.vue`：租户控制台左侧导航（工作面板/镜像市场/设置…）
- 路由 `tenantRoutes.js`：`/tenant/:tenant/workspace/:id/settings/...` 工作空间级页面先例

## 设计决策

### D1 — 入口位置：租户控制台左侧导航（Sidebar）
「工作空间左边」= 工作面板所在页面的左侧导航（Sidebar.vue，唯一左侧栏）。新增一级导航项「工作空间的排队调度」，点击进入新页面 `/tenant/:tenant/queue-schedule/`（工作空间由页面解析：URL `workspace_id` 查询参数 > /me/ `current_workspace` > 默认工作空间；页面内提供 WorkspaceSwitcher 切换，对齐 WorkPanel OPT-20260816-047 约定）。

### D2 — 数据模型：工作空间级节奏（新增，不改旧表）
新增两表（dataMigrate/taskTaskService/014_workspace_schedule_rhythms.sql）：
- `workspace_schedule_rhythms(workspace_id PK, tenant_id, enabled, timezone, updated_at)`
- `workspace_schedule_rhythm_windows(id PK, workspace_id, tenant_id, daily_start, daily_end, max_queued_machines, auto_close, auto_close_warn_minutes, auto_close_warn_key, auto_close_release_key, sort_order, created_at, updated_at)`

旧 task 级表保留不迁移（expand/contract 的 expand 阶段）。校验逻辑（HH:MM、交叠检测）复用既有 helpers。

### D3 — API（契约优先）
- `GET /api/tenant/{tid}/workspace/{wid}/queue-schedule/` → 200：`{ workspace_id, schedule_rhythm:{enabled,timezone,windows:[...]}, in_window, window_message, queued_slots_used, max_queued_machines, members:[{task_id,title,status,enqueued_at,deferred_reason,...}] }`（成员带 title，工作空间内全队列）
- `PUT /api/tenant/{tid}/workspace/{wid}/queue-schedule/` body `{ enabled, timezone, windows:[{id,daily_start,daily_end,max_queued_machines,auto_close,auto_close_warn_minutes}] }` → 200（校验：HH:MM、交叠、顶层约束不适用）
- 鉴权：`requireTenantMember`（main.go 全局）+ `hasWorkspaceAccess(tenantID, wid, userID)`（对齐 handleGetQueuedAutoRun）
- 错误格式沿用 `writeError`（既有约定）
- 旧端点 GET `/todos/{taskId}/queued-auto-run/` 与 PATCH `schedule_rhythm` 保留（向后兼容，UI 不再使用）

### D4 — 调度分发：工作空间级 + 旧行为回退
`runQueuedScheduleDispatchOnce` 改为按 `workspace_id` 分组：
- **工作空间已配置节奏**（workspace_schedule_rhythms 有行）：
  - 未启用 → 该工作空间全部成员 deferred（reason「节奏未启用」）
  - 不在窗口 → 全部 deferred（reason「等待时段 …」）
  - 在窗口 → 按 `depth ASC, enqueued_at ASC` 依序启动成员至 maxN（在窗口窗口的 max_queued_machines 之和；`starting` 跳过），启动逻辑复用 `startQueuedMembership`（逐个 = 每个分发周期启动空余槽位数）
- **工作空间未配置节奏** → 回退既有 top_task 级 `dispatchTopQueue`（生产既有配置不中断；用户以新 UI 配置工作空间节奏后接管）

### D5 — 入队状态与展示
- `enqueueQueuedAutoRun` 初始 status：有效节奏 = 工作空间节奏（存在时）否则 top-task 节奏（legacy）；未启用/不在窗口 → deferred
- taskJSON `queued_deferred_reason` 同理走有效节奏
- `handleGetQueuedAutoRun`（旧端点）保留原语义不动（仍按 top_task），新页面走新端点

### D6 — 自动关闭（auto_close）
`runQueuedScheduleAutoCloseOnce` 增加工作空间节奏分支（`enabled=1 AND auto_close=1` 的 workspace 窗口 → 按工作空间扫成员机器停止 + 清槽位）；旧 task 级分支保留。`runAutoCloseForWorkspace` 镜像 `runAutoCloseForTop`。

### D7 — 前端任务详情精简
`TaskDetailTaskAuxInfoPanel.vue` 不再嵌入完整面板。新增紧凑组件 `TaskDetailQueuedScheduleToggle.vue`：仅「加入调度/退出调度」切换 + 状态 chip（复用 useQueuedAutoRunPanel 的 join/leave/status，不含节奏表单）。旧 `TaskDetailQueuedSchedulePanel.vue` 删除（拆出）。

### D8 — 新页面 WorkspaceQueueSchedule.vue
路由 `/tenant/:tenant/queue-schedule/`（components: Navbar+Sidebar）。内容：
- 标题「工作空间的排队调度」+ WorkspaceSwitcher
- 节奏设置卡（enabled 开关、时间窗列表增删、每日起止、并发机器数、自动关闭、时区、交叠校验、保存）— UI 复用 TaskDetailQueuedSchedulePanel 表单样式与校验逻辑（提取共用函数或复制小函数，Rule 0 极简优先）
- 调度状态卡（in_window/window_message/占用槽位）
- 队列列表卡（成员按序：序号、title、状态 chip、deferred_reason）
- 数据加载：GET 新端点；保存：PUT 新端点
- 新 composable `useWorkspaceQueueSchedule.js`（快照加载 + 保存 + join/leave 由任务详情负责）

### D9 — 可观测性
- 工作空间节奏保存/分发沿用既有 `tracelog.LogForwardStage` + domain events（`WorkspaceScheduleSaved`、复用 `QueuedAutoRunDeferred/Started`、`ScheduleAutoCloseReleased`），日志含 workspace_id，禁止密钥/PII

### D10 — 测试策略
- Go：`queued_schedule_workspace_test.go`（窗口校验复用、节奏 upsert、GET/PUT handler、dispatch 工作空间级：未配置回退/未启用 deferred/窗口内依序启动/槽位上限）
- 前端：`WorkspaceQueueSchedule.test.js`（表单加载/保存/校验）、`TaskDetailQueuedScheduleToggle.test.js`（加入/退出）、`useWorkspaceQueueSchedule.test.js`
- Playwright：改造 `TaskDetail.schedule-rhythm-save-success` 等（设置入口移至新页面）；既有 panels-regression 维护

## 风险与兼容
- 旧 task 级节奏在「工作空间未配置」时继续生效（回退），新 UI 一旦保存工作空间节奏即整体接管 → 需在页面提示「保存后以工作空间节奏为准」
- 生产已有排队任务：回退逻辑保证状态机不重置；窗口语义不变（daily HH:MM，Asia/Shanghai 默认）
- 删除 TaskDetailQueuedSchedulePanel 前确认无其他引用（仅 TaskDetailTaskAuxInfoPanel）
