# 设计：排队调度节奏「是否自动关闭」

- 日期：2026-07-22
- 状态：已采纳（goal-mode 自动决策，跳过确认门）
- 架构版本：v44 🎯 target
- `python_api_approval`: scoped-down（**零新增 Python HTTP 接口**；落 Go TTS/Cloud/Gateway + Vue + onlineServiceJS）
- 相关：`top_deliverable_queued_auto_run_schedule`（v40）、容器 `task-lifecycle/shutdown`

## 1. 问题

排队窗口结束后，由调度拉起的机器仍可能继续占用；缺少「窗口结束前预告 + 到期自动释放」能力，成本与资源回收不可控。

## 2. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 顶层节奏可配置 `auto_close`（默认 false） | PATCH/GET 回显；UI 勾选 |
| S2 | 勾选且窗口启用时：距 `daily_end` ≤5 分钟且仍在窗内 → 通知占用槽位对应容器 | 单测 + 事件 |
| S3 | 窗外且仍有 `queued_machine_slots` → 自动下发释放（优先容器 shutdown，失败回退 Cloud `stop-vm`） | 单测 + 事件 |
| S4 | 同一窗口周期 warn/release 各最多一次（幂等） | 单测 |
| S5 | 无新 Python HTTP 接口 | Go + Vue + onlineServiceJS |

## 3. 方案对比（自动采纳 A）

| 方案 | 描述 | 取舍 |
|------|------|------|
| **A（采纳）** | TTS Dispatcher 扩展：读 `auto_close`；T-5min 经 Cloud→Gateway 调容器 `closing-soon`；窗外调 `shutdown`/`stop-vm` | 与现有节奏同进程，槽位 SSOT 清晰 |
| B | 容器自轮询窗口结束 | 容器缺权威窗口配置，时钟漂移难控 |
| C | 新建 scheduler 微服务 | MVP 过度 |

### 已锁定产品决策

| 决策 | 取值 |
|------|------|
| 配置挂载 | 顶层 `schedule_rhythm.auto_close` |
| 预告提前量 | **固定 5 分钟**（MVP 不可配置） |
| 作用范围 | 仅该顶层下 `queued_machine_slots` 中的任务机器（排队调度拉起） |
| 预告动作 | `POST …/container-task-lifecycle-closing-soon` → 容器 `/task-lifecycle/closing-soon` |
| 到期释放 | 优先 `container-task-lifecycle-shutdown`（容器内 `request-machine-release`）；失败则 Cloud `stop-vm`；清理槽位 |
| `auto_close=false` | 不预告、不自动释放（窗外仍只 deferred） |

## 4. 领域概念

| 概念 | 说明 |
|------|------|
| **ScheduleRhythm.auto_close** | 窗口到期自动关闭开关 |
| **AutoCloseWarnKey / ReleaseKey** | 幂等键：窗口结束日的 `YYYY-MM-DD` + `daily_end` |
| **QueuedScheduleAutoCloser** | Dispatcher 旁路：warn → release |

## 5. 数据模型（TTS）

```sql
ALTER TABLE top_deliverable_schedule_rhythms ADD COLUMN auto_close INTEGER NOT NULL DEFAULT 0;
ALTER TABLE top_deliverable_schedule_rhythms ADD COLUMN auto_close_warn_key TEXT NOT NULL DEFAULT '';
ALTER TABLE top_deliverable_schedule_rhythms ADD COLUMN auto_close_release_key TEXT NOT NULL DEFAULT '';
```

JSON：`schedule_rhythm.auto_close: bool`

## 6. API / 运行时

- 既有 `PATCH/GET .../todos/{id}/` 扩展 `schedule_rhythm.auto_close`
- Cloud：`isContainerOutboundComputeSub` 增加 `compute/container-task-lifecycle-` 前缀
- Gateway L0：`container-task-lifecycle-closing-soon` → `/task-lifecycle/closing-soon`
- onlineServiceJS：新增 `POST /api/task-lifecycle/closing-soon`（记录预告、不释放）
- Dispatcher ticker（~30s）在 `dispatchTopQueue` 前后调用 `runAutoCloseForTop(topID)`：
  1. `!auto_close` → return
  2. 在窗且 `0 < minutesUntilEnd ≤ 5` 且 warn_key 未写 → 通知各槽位任务容器 → 写 warn_key → 事件
  3. 不在窗且有槽位且 release_key 未写 → shutdown/stop-vm → 清槽位 → 写 release_key → 事件

## 7. 业务意图 → 事件

| 业务意图 | 事件名 | 发布点 |
|---------|--------|--------|
| 更新节奏（含 auto_close） | ScheduleRhythmUpdated | TTS PATCH |
| 窗口结束前预告容器 | ScheduleAutoCloseWarned | TTS auto-closer |
| 窗口结束自动释放机器 | ScheduleAutoCloseReleased | TTS auto-closer |

## 8. 架构变更

- 视图：application-integration **v44**（基于 v41 current）
- 交付：`.puml` + `.archimate` + `.mermaid.md` + VERSION_HISTORY

## 🕸️ Code Review Graph 分析

- **graph_status**: stale/sparse（CLI `uvx code-review-graph`；MCP server 未挂载；图仅 ~11 files / 70 nodes，未覆盖 TTS Go）
- **探测路径**: MCP unavailable → CLI `update`/`search`/`status`
- **触点社区 / 关键节点**: 既有排队调度落点（文件级）`taskTaskService/src/queued_schedule*.go`、`TaskDetailQueuedSchedulePanel.vue`、`taskLifecycleShutdown.mjs`、`container_gateway_proxy.go`、`l0_registry.go`
- **爆炸半径（文件/函数）**: `applyScheduleRhythmFromBody`、`dispatchTopQueue`/`startQueuedScheduleTicker`、`rhythmToJSON`、Cloud outbound proxy 前缀、Gateway L0、容器 lifecycle 路由
- **与 Archimate 对照**: 扩展 v40 节奏调度边（TTS→Cloud→Gateway→容器；TTS→stop-vm）
- **风险与约束**: 幂等键防重复释放；容器不可达时必须有 stop-vm 回退；勿误伤 `started_via!=queued_schedule` 的手动机
- **对本次设计的影响**: 采用槽位表作为释放集合 SSOT；不扫描全 workspace CSC

## 9. 非目标

- 可配置预告分钟数
- 对立即 auto_run / 手动启服机器的自动关闭
- 新建独立调度微服务
