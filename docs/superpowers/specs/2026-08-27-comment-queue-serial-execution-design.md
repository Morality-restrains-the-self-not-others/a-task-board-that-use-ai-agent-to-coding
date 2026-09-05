# 任务详情：自动执行队列 + 评论逐条执行 — 设计

- 日期：2026-08-27
- 入口：/goal（跳过 USER GATE）
- 范围：taskFE 任务详情页 UI 组合（无新 API / 无 schema / 无架构变迁）
- 页面：云端开发任务详情「添加评论」区

## Context

用户在任务详情同时点选了两块互不相关的控件：

1. **任务辅助信息**内的「加入/离开队列」+ 状态「自动调度未启用」
2. **添加评论**区的「执行依赖」（等待前序 / 不等待前序 / 自动提交）

两者都在讲「按顺序执行」，但分层不同、位置割裂：

| 控件 | 实际语义 | 问题 |
|------|----------|------|
| 队列 toggle | 工作空间级：本**任务**何时轮到启服 | 藏在可折叠「辅助信息」，与发评论决策脱节 |
| 执行依赖 | 评论级：本**评论**是否等前序 agent | 用户无法把「入队」理解成「评论一条一条跑」 |

「自动调度未启用」只说明工作空间时段未开，并不解释入队后评论如何串行。用户期望：**启用自动排列/调度时，评论一条一条执行**，两块必须结合设计。

## Decision

**We will** 把队列 toggle 迁入评论「执行依赖」同一卡片，并在入队后锁定串行依赖。

1. `CommentExecutionDependencyPicker` 在任务详情 composer 中展示「自动执行」区块：队列加入/离开 + 时段入口 + 逐条执行说明 + 原执行依赖 + 自动提交。
2. `task.queued_auto_run === true` 时强制 `wait_previous`（等前面全部评论为默认范围），禁用「不等待前序」。
3. 任务辅助信息不再放队列 toggle（折叠辅助信息不再藏起入队入口）。
4. 看板卡片 composer 仍 `showDependencyPicker=false`，不出现队列。
5. 入队/出队仍走既有 `PATCH .../todos/{id}/` `{ queued_auto_run }` + `Idempotency-Key`；`task-updated` 从 picker → composer → comments → TaskDetail `onServerConfigTaskUpdated`。

### 文案

- 未入队：`加入自动执行队列后，本任务的评论会按提交顺序一条一条执行（需先启用工作空间「自动调度」）。`
- 已入队：`已加入自动执行队列：本任务轮到后，评论按提交顺序一条一条执行（等前序完成）。`

### 非目标

- 不改工作空间调度分发、评论 binding 调度后端
- 不为入队批量 PATCH 历史评论的 `execution_mode`（仅约束**新发评论**草稿；已发出评论仍可在执行细节改依赖）
- 无新 HTTP 端点 → Step 2 角色权限 SKIP
- 无新聚合/事件 → Step 6 DDD 书面例外：纯前端展示与草稿约束；领域事件仍由既有 PATCH queued_auto_run / CommentExecutionModeChanged 路径发布

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 仅加文案、位置不动 | 两控件仍分属辅助信息与评论区，折叠后入队入口消失 |
| 入队后隐藏整个依赖选择器 | 丢失「等指定评论」与自动提交 |
| 入队时 PATCH 全部历史评论为 wait_previous | 超出页面调整范围，可能打断已选 independent 的并行评论 |

## NFR（嵌入，无独立文件）

### 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 动作 |
|------|---------|------|------|
| `PATCH /api/tenant/{tid}/workspace/{wid}/todos/{taskId}/` | tenant + workspace + taskId | 已有合适分片键 | 无 |
| 任务详情路由 `/tenant/:tenant/workspace/:ws/task-detail/:task/` | 同上 | 已有 | 无 |

全部路径已携带可分片 ID，L0 升级触发：出现跨任务列表写操作时再评估。

### 幂等性强制审视

| 路径 | 副作用 | 重复触发 | 幂等键 | 重放语义 |
|------|--------|----------|--------|----------|
| 加入/离开队列 PATCH | 写 `queued_auto_run` | 连点按钮 | `Idempotency-Key`（既有 `createClickGuard`） | 同键重试同一意图 |
| 发评论（既有） | 写评论 + 调度 | 提交按钮 | 既有 composer clickGuard | 不变 |
| 依赖 radio | 仅本地草稿 | 无服务端写 | Anti-Replay-OK: 本地 v-model | — |

资金/云资源不在本增量。NFR 级别 L2。

## Value stream（最小增量）

用户打开任务详情 → 在「添加评论」看到自动执行与执行依赖同卡 → 加入队列 → 依赖锁定等待前序 → 提交评论 → 评论按前序串行调度（既有 wait_previous / waiting_previous）。

## Consequences

- 入队入口始终在发评决策旁，不再被辅助信息折叠藏起
- 「不等待前序」在入队期间不可选，避免与队列「一条一条」语义冲突
- 看板卡片不受影响
