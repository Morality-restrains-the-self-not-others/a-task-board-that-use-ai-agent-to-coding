# 设计：顶层交付物排队自动执行调度节奏

- 日期：2026-07-19
- 状态：已采纳（goal-mode 自动决策，跳过确认门）
- 架构版本：v40 🎯 target
- `python_api_approval`: scoped-down（**零新增 Python HTTP 接口**；全部落 Go + Vue）
- 相关：`auto_run` 立即启服、`workspace_machine_idle_policy`（闲置复用/回收，额度正交）

## 1. 问题

1. 顶层交付物（如价值流）下大量子任务若全部立即 `auto_run`，无法利用低价时段，成本高。
2. 缺少「按顶层任务配置时段 + 排队并发机器数、子树统一排队」的能力。
3. 现有 `auto_run` 为 fire-and-forget goroutine，无队列、无窗外 deferred 态。

## 2. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 顶层交付物任务（`parent_task_id` 空）可配置每日窗口 + 排队并发上限 | PATCH/GET 回显 |
| S2 | 子树任务新开关 `queued_auto_run` 加入排队；手动启服出队 | 单测 + API |
| S3 | 排队并发只统计调度器拉起的机器，与立即 auto_run/手动互不干扰 | `started_via=queued_schedule` |
| S4 | 窗外成员 `deferred`，UI 可展示等待时段 | 状态字段 + 队列 API |
| S5 | 出队顺序：相对顶层深度升序，同层 FIFO | 调度单测 |
| S6 | 无新 Python HTTP 接口 | Go TTS + Cloud + Vue |

## 3. 方案对比（自动采纳 A）

| 方案 | 描述 | 取舍 |
|------|------|------|
| **A（采纳）** | TTS 拥有节奏配置与排队成员表；ticker 调度；代调 Cloud `start-vm-auto` 并标 `started_via` | 与任务树同进程；事件易发 |
| B | Cloud 拥有队列 | 任务树/深度属 TTS，跨服务读父链重 |
| C | 新建 scheduler 微服务 | MVP 过度 |

### 已锁定产品决策

| 决策 | 取值 |
|------|------|
| 配置挂载 | 顶层交付物**任务实例** |
| 并发口径 | 仅「排队调度拉起」的机器数 |
| 入队方式 | 新开关 `queued_auto_run` |
| 手动执行 | 出队（再次入队需重开开关） |
| 窗外 | `deferred` + UI 说明窗口 |
| 排序 | 深度升序（先近后远），同层 FIFO |
| 时段 MVP | 每日固定窗口（可跨午夜）+ 时区 |
| 边角 A | 顶层任务本身可入队（depth=0） |
| 边角 B | 节奏 disabled → 成员保持 deferred，不回退立即 auto_run |

## 4. 领域概念（轻量 → Step 6）

| 概念 | 说明 |
|------|------|
| **ScheduleRhythm** | 属顶层 Task：enabled、daily_start/end、timezone、max_queued_machines |
| **QueuedAutoRunMembership** | 入队成员：top_task_id、depth、enqueued_at、status |
| **QueuedScheduleDispatcher** | 领域服务：窗口判定、额度、BFS 出队、触发启服 |
| **StartedVia** | Cloud 侧启服来源：`queued_schedule` \| `auto_run` \| `manual` \| … |

## 5. 数据模型

### taskTaskService（实现：独立表，避免改 tasks 宽扫描）

```sql
CREATE TABLE top_deliverable_schedule_rhythms (
  task_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 0,
  daily_start TEXT NOT NULL DEFAULT '', -- HH:MM
  daily_end TEXT NOT NULL DEFAULT '',
  timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
  max_queued_machines INTEGER NOT NULL DEFAULT 0,
  updated_at DATETIME NOT NULL
);

CREATE TABLE queued_auto_run_memberships (
  task_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  top_task_id TEXT NOT NULL,
  depth INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'queued', -- queued|deferred|starting
  enqueued_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE queued_machine_slots (
  task_id TEXT PRIMARY KEY,
  top_task_id TEXT NOT NULL,
  acquired_at DATETIME NOT NULL
);
```

表所有权：上述三表 → `task-task-service`。API 仍用 `schedule_rhythm` / `queued_auto_run` JSON 字段。

### taskCloudService

```sql
ALTER TABLE cloud_server_configs ADD COLUMN started_via TEXT DEFAULT '';
-- queued_schedule | auto_run | manual | reuse | ''
```

占用查询：同 workspace 下 `started_via='queued_schedule'` 且 runtime 为 starting/started 的 config 数；或 TTS 维护 `starting` 成员 + Cloud 查询 running。

## 6. API（全部 Go）

前缀任务：`/api/tenant/{tenant}/workspace/{workspace}/todos/`

| 方法 | 路径 | 说明 |
|------|------|------|
| PATCH | `{id}/` | 顶层可写 `schedule_rhythm`；任意任务可写 `queued_auto_run` |
| GET | `{id}/` | 回显节奏、排队状态、窗口文案 |
| GET | `{top_id}/queued-auto-run/` | 队列快照：成员排序、占用/上限、当前是否在窗口 |

Cloud：`start-vm` / `start-vm-auto` body 可选 `started_via`；缺省按调用方推断（TTS 排队路径传 `queued_schedule`）。

权限：与现有 todos 一致——工作空间可写成员可改节奏/排队；只读成员可看队列。

## 7. 运行时

```
queued_auto_run=true
  → resolve top ancestor with schedule (or self if top)
  → upsert membership (depth, status=queued|deferred)
  → event TaskQueuedForAutoRun

dispatcher ticker (TTS, ~30s):
  for each distinct top_task_id with memberships:
    if !rhythm.enabled → mark deferred ("节奏未启用")
    else if !inWindow → mark deferred ("等待时段")
    else:
      slots = max - countQueuedMachines(top)
      pick next by (depth ASC, enqueued_at ASC) where status in (queued,deferred)
      for up to slots: status=starting → start-vm-auto(started_via=queued_schedule)
      → event QueuedAutoRunStarted

manual start-vm / user start:
  → dequeue membership + queued_auto_run=false (or keep flag false + leave queue)
  → event TaskDequeuedFromAutoRun
```

## 8. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 | 例外 |
|---------|--------|--------|--------------|------|
| 更新顶层调度节奏 | ScheduleRhythmUpdated | TTS PATCH | SSE/看板可选 | — |
| 加入排队 | TaskQueuedForAutoRun | TTS | 入队 | — |
| 离开排队 | TaskDequeuedFromAutoRun | TTS | 出队 | — |
| 窗外/未启用等待 | QueuedAutoRunDeferred | TTS dispatcher | UI deferred | — |
| 调度启服成功 | QueuedAutoRunStarted | TTS | 占额度 | — |
| 查询队列快照 | — | — | — | 纯查询，无事件 |

## 9. 前端

- 任务详情默认**不**常驻展开节奏表单；展示入口按钮「排队调度设置」，点击后以**模态框**编辑
- 模态内：顶层可配置调度节奏（启用、起止时间、时区、最大排队机器数）；任意任务可开关 `queued_auto_run`
- 入口旁/模态内徽章：排队中 / 等待时段 / 调度启服中
- 顶层：简易队列列表（深度、入队时间、状态）—— MVP 可后续补（见 OPT 队列快照）

## 10. 价值流影响（粗）

- 新 step：`top-deliverable-queued-auto-run-schedule`（task-management）
- 并列既有 `create-task-auto-run-backend-start`，不替换
- 字段：`task.tasks.schedule_rhythm_*`、`task.tasks.queued_auto_run`、`task.queued_auto_run_memberships.*`、`cloud.cloud_server_configs.started_via`

## 11. 🏛️ 架构变更影响

- **迭代版本**: v40 🎯 target
- **迭代名称**: top-deliverable-queued-auto-run-schedule
- **作者**: claude
- **设计日期**: 2026-07-19 17:46
- **新增文件**（每个视图三类伴生，缺一不可）:
  - 🆕 `docs/architecture/v40-application-integration-20260719-1746-claude.puml`
  - 🆕 `docs/architecture/v40-application-integration-20260719-1746-claude.archimate`（含 Plateau/Gap/WP）
  - 🆕 `docs/architecture/v40-application-integration-20260719-1746-claude.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskTaskService — ScheduleRhythm / Queue / Dispatcher
  - 🟡 [MODIFIED] taskCloudService — `started_via`
  - 🟡 [MODIFIED] Vue 任务详情 — 节奏与排队 UI
  - 🟢 [NEW] DataObject `queued_auto_run_memberships`

### .archimate 架构变迁要点

| 元素 | 内容 |
|------|------|
| Plateau v39 | people-member-group-go current |
| Plateau v40 | 顶层交付物排队调度 |
| Gap | 无顶层子树时段排队与隔离并发 |
| WorkPackage | WP-v40-queued-auto-run-schedule |

## 12. 非目标（MVP 不做）

- 每周多段 / 节假日日历
- 跨顶层全局公平调度
- 优先级字段
- Python 新接口
