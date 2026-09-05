# 自动调度安排 · 调度历史 — 设计文档

- 日期：2026-08-28
- 入口：/goal（跳过 USER GATE）
- 页面：`/tenant/:tenant/queue-schedule/?workspace_id=`（壳标题「云端开发」）
- 元素：`data-testid="schedule-status-bar"`（当前时段/槽位快照条）

## 架构基线理解

当前架构 **v114 current**（企业景观 / 应用集成）。本增量在既有 taskFE → APISIX → taskTaskService 链上新增查询模型与只读 API，不拆服务。

- 业务层：工作空间用户查看「自动调度安排」现状（时段内/外、槽位）
- 应用层：`WorkspaceQueueSchedule` 只渲染 GET 快照；分发/入队事件已 `publishDomainEvent`，**无持久化查询面**
- 技术层：`workspace_schedule_rhythms` + `task_queued_auto_run_memberships` 仅当前态

CRG：`code-review-graph update --brief` 成功。无 `codegraph_explore` MCP；基线来自 `codegraph query` + 源码。

TraceId：用户输入无 `data-traceId`，跳过 Loki。

## 🕸️ Code Review Graph 分析

- 符号：`handleWorkspaceQueueScheduleRoutes`、`useWorkspaceQueueSchedule`、`dispatchWorkspaceQueueWith`、`enqueueQueuedAutoRun`
- 爆炸半径：TTS 排队调度包 + taskFE 调度页；网关已有 `/queue-schedule/*` 前缀，history 子路径无需新 APISIX 条目
- 社区：task 协作 / 工作空间调度

## 问题

状态栏只回答「此刻」：○ 不在允许运行时段、等待 22:30–23:30、占用 0/1。用户无法回答「刚才有没有拉起过任务、何时进入等待、谁改过时段」。membership 只有 `status/enqueued_at/updated_at`，窗口切换与启服轨迹不可查。

## 方案（采用）

**append-only 工作空间调度历史表 + 状态栏下方时间线卡片。**

### 读模型

表 `task_queued_schedule_history`（taskTaskService / `task_task`）：

| 列 | 说明 |
|---|---|
| id | `qsh_<snowflake>` VARCHAR(64) |
| tenant_id / workspace_id | 分片键 |
| event_type | 见下表 |
| task_id | 可空；任务级事件填写 |
| message | 人读一句（含时段文案/任务标题快照） |
| actor_user_id | 用户操作填写；timer 为空 |
| created_at | UTC；分区键 |

主键 `(id, created_at)`，按月 RANGE 分区（策略 A），查询索引 `(workspace_id, created_at)`。热窗 90 天；冷分区由 `ensure_partitions.sh` TRUNCATE。

**不记录** timer 空扫。只在状态真实变化时 append（fail-open，不阻断主路径）。

### event_type

| type | 何时 | 粒度 |
|---|---|---|
| `rhythm_saved` | PUT 节奏成功 | 工作空间 |
| `member_enqueued` | 入队 | 任务 |
| `member_dequeued` | 出队 | 任务 |
| `member_started` | 调度 start-vm 成功 | 任务 |
| `window_entered` / `window_exited` | 分发检测 in_window 翻转 | 工作空间（避免窗外每人一条 deferred 刷屏） |
| `auto_close_warned` / `auto_close_released` | 既有 auto-close | 工作空间 |

窗外 per-member `QueuedAutoRunDeferred` 仍发领域事件，**不**写入历史表（由 `window_exited` 概括）。入队当时即 deferred 仍记 `member_enqueued`，message 带等待原因。

窗口翻转：`workspace_schedule_rhythms` 增加 `last_in_window`（nullable TINYINT）。首次分发写入 entered/exited；之后仅翻转时写。

### API

既有 GET 快照增加可选字段 `recent_history`（最多 8 条，新→旧），向后兼容。

新只读：

```
GET /api/tenant/{tid}/workspace/{wid}/queue-schedule/history/?limit=20&cursor=
```

鉴权与 GET 快照相同：`hasWorkspaceAccess`。分页 cursor = `{created_at}|{id}`；响应 `{ items, next_cursor, has_more }`。错误格式沿用 TTS `writeError`。

网关：`workspace-queue-schedule-direct` 已覆盖 `queue-schedule/*`。TTS `parts[4]=="history"` 分发 GET。

OpenAPI：`taskTaskService/src/openapi.yaml` 显式登记 history 与 GET 新增字段。

### UI

保留状态栏。其下新卡 `data-testid="schedule-history-card"`：

- 标题「调度历史」
- 行：本地时间 · 类型徽章 · message · 任务则真实 `a[href]`
- 空态：「尚无调度记录。入队、改时段或自动启服后会出现在这里。」
- 「加载更多」→ history GET；Anti-Replay-OK: 只读分页
- 初始用 snapshot.recent_history；刷新走既有 `reload()`
- **禁止** setInterval 轮询

抽出 `ScheduleHistoryCard.vue`（页面已近 400 行）。

### 事件

| 业务意图 | 事件名 | 发布点 | 消费者 |
|---|---|---|---|
| 保存工作空间节奏 | WorkspaceScheduleSaved | 既有 PUT | 本增量 append `rhythm_saved` |
| 入队 | TaskQueuedForAutoRun | 既有 | append `member_enqueued` |
| 出队 | TaskDequeuedFromAutoRun | 既有 | append `member_dequeued` |
| 调度启服 | QueuedAutoRunStarted | 既有 | append `member_started` |
| 进入允许时段 | WorkspaceScheduleWindowEntered | **新** dispatch 翻转 | append `window_entered` |
| 离开允许时段 | WorkspaceScheduleWindowExited | **新** dispatch 翻转 | append `window_exited` |
| auto-close 预告/释放 | ScheduleAutoCloseWarned/Released | 既有 | append 对应 type |
| 查询历史 | — | GET | 纯查询 |

## 拒绝的方案

1. **只从 membership.updated_at 拼时间线** — 丢失翻转与启服，无法满足「调度历史」。
2. **Kafka/Loki 当查询面** — 延迟与租户隔离不适合产品页。
3. **每轮 deferred 落一行** — 窗外每 30s 扫描会对全队列刷屏。
4. **独立 Kafka 消费者写历史** — 同 BC 内同步 append 更简单，避免消费滞后。

## 🐍 Python 新增接口

not_applicable — 全部 Go taskTaskService。

## 测试

- Go：insert/list 分页、GET 403 无 workspace、快照含 recent_history、入队/保存/窗口翻转各写一行、空扫不写
- FE：状态栏仍在；历史卡渲染；加载更多带 cursor；空态；错误 `data-traceId`

## 架构变更

v115 target：新 DataObject `task_queued_schedule_history`；GET history；事件 WindowEntered/Exited。
