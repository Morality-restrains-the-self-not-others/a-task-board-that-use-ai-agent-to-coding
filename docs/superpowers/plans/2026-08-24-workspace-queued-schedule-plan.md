# 工作空间级排队调度 — 实施计划

- 日期：2026-08-24

## 切片总览（垂直切片优先）

### 切片 A（后端基础）— dataMigrate + Go 数据层
1. `dataMigrate/taskTaskService/014_workspace_schedule_rhythms.sql`：两新表（IF NOT EXISTS + 索引）
2. `taskTaskService/src/queued_schedule_workspace.go`：workspaceScheduleRhythm/Window 结构 + load/upsert + `applyWorkspaceScheduleRhythmFromBody`（复用 parseHHMM/validateWindowsNoOverlap）+ 有效节奏解析 helper（workspace 优先、task 回退）
3. 测试：`queued_schedule_workspace_test.go`（upsert/加载/校验/交叠/回退解析）

### 切片 B（后端 API）— 路由 + handlers
4. main.go `/api/tenant/{tid}/workspace/{wid}/queue-schedule` 分支（GET/PUT，requireTenantMember 已有 + hasWorkspaceAccess）
5. `queued_schedule_workspace_handlers.go`：handleGetWorkspaceQueueSchedule（快照含成员 title JOIN）+ handlePutWorkspaceQueueSchedule（保存 + WorkspaceScheduleSaved 事件 + tracelog）
6. 测试：handler 测试（GET 200/403、PUT 校验失败 400、保存后 GET 一致）

### 切片 C（后端分发）— 工作空间级调度
7. `queued_schedule_dispatch.go`：runQueuedScheduleDispatchOnce 按 workspace 分组 → dispatchWorkspaceQueue（节奏存在→工作空间逻辑；不存在→回退 dispatchTopQueue）
8. `queued_schedule.go`：enqueueQueuedAutoRun 初始状态与 taskJSON deferred_reason 走有效节奏（workspace→task 回退）
9. `queued_schedule_auto_close.go`：runQueuedScheduleAutoCloseOnce 增加 workspace 分支 + runAutoCloseForWorkspace
10. 测试：分发回退/未启用/窗口内依序启动/槽位上限/auto_close workspace 分支

### 切片 D（前端数据层）— composable + API
11. `taskFE/app/src/composables/workspaceSchedule/useWorkspaceQueueSchedule.js`：快照加载 + 保存 + 有效状态（in_window/window_message/slots）
12. 测试：useWorkspaceQueueSchedule.test.js

### 切片 E（前端页面）— WorkspaceQueueSchedule.vue
13. 路由 `/tenant/:tenant/queue-schedule/`（tenantRoutes.js，Navbar+Sidebar 壳）+ 视图（标题「工作空间的排队调度」、WorkspaceSwitcher、节奏表单卡、调度状态卡、队列列表卡）
14. 测试：WorkspaceQueueSchedule.test.js（表单加载/保存/校验/队列渲染）

### 切片 F（前端入口+精简）— Sidebar + 任务详情
15. Sidebar.vue 新增导航项「工作空间的排队调度」
16. 新组件 TaskDetailQueuedScheduleToggle.vue（加入/退出 + 状态 chip）替换 TaskDetailTaskAuxInfoPanel 中的完整面板；删除 TaskDetailQueuedSchedulePanel.vue
17. 测试：TaskDetailQueuedScheduleToggle.test.js + 更新 TaskDetailQueuedSchedulePanel.test.js（改为 toggle 语义）；Playwright：schedule-rhythm-save-success 改指向新页面（或改断言）

### 切片 G（收尾）
18. 全量回归（taskTaskService go test、taskFE vitest、Playwright 子集）+ 提交（原子，逐切片已提交）+ 推送

## 事件契约 → 投递清单
- WorkspaceScheduleSaved（新）→ publishDomainEvent（TTS 内部通道，与既有一致）
- 其余复用既有事件

## 关键文件
- 新：dataMigrate/taskTaskService/014_workspace_schedule_rhythms.sql
- 新：taskTaskService/src/queued_schedule_workspace.go、queued_schedule_workspace_handlers.go、（+2 测试文件）
- 改：taskTaskService/src/queued_schedule_dispatch.go、queued_schedule.go、queued_schedule_auto_close.go、main.go
- 新：taskFE/app/src/views/WorkspaceQueueSchedule.vue、composables/workspaceSchedule/useWorkspaceQueueSchedule.js、components/task-detail/TaskDetailQueuedScheduleToggle.vue
- 改：taskFE/app/src/components/Sidebar.vue、TaskDetailTaskAuxInfoPanel.vue、router/tenantRoutes.js
- 删：taskFE/app/src/components/task-detail/TaskDetailQueuedSchedulePanel.vue
