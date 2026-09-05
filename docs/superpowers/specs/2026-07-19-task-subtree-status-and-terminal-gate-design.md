# 设计：任务详情子树状态展示与终态门禁

- 日期：2026-07-19
- 状态：已采纳（goal-mode 自动决策，跳过确认门）
- 架构版本：v41 🎯 target
- `python_api_approval`: n/a（**零新增 Python HTTP 接口**；全部落 Go TTS + Vue）
- 相关：`parent_task` 任务树、进度列终态（已完成/已取消）、`TASK_STATUS_CHANGED`

## 1. 问题

1. 任务详情页（`/tenant/.../workspace/.../task-detail/...`）仅展示上层交付物，**不展示子任务/孙子任务**的当前进度与完成情况，父任务负责人无法在详情内对齐子树进度。
2. 将任务标记为「已完成」或「已取消」时，**未校验**子任务/孙子任务是否已进入完成或取消终态，可导致父任务先关闭而子树仍开放。

## 2. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 详情页展示直接子任务与孙子任务（depth≤2）的标题、进度列名、完成标志；无子树时隐藏区块或显示空态 | UI + 单测 |
| S2 | 展示完成情况摘要：`settled/total`（settled = 已完成∪已取消） | API + UI |
| S3 | 父任务 PATCH 进入终态（进度列「已完成」/「已取消」，或 `completed` false→true）时，**全部后代**（任意深度）须已为终态，否则 **409** 并返回未关闭任务列表 | Go 单测 |
| S4 | 前端进度变更失败展示可读错误且带 `data-traceId`；看板拖拽同路径复用服务端门禁 | 前端单测 |
| S5 | 无新 Python HTTP 接口；事件：成功终态仍发 `TASK_STATUS_CHANGED`；门禁拒绝不发事件 | 意图对照表 |

## 3. 方案对比（自动采纳 A）

| 方案 | 描述 | 取舍 |
|------|------|------|
| **A（采纳）** | TTS 新增 `GET .../todos/{id}/subtree/`；PATCH/`/switch` 写路径加 `DescendantTerminalGate`；详情 UI 新面板 | 数据所有权在 TTS；单一真相；看板/详情同源门禁 |
| B | 纯前端用 workspace todos 过滤 + 仅前端拦截 | 可绕过；大工作区列表重；无服务端保证 |
| C | Django 校验 | 违反 Go-first；任务表非 Django owner |

### 已锁定产品决策

| 决策 | 取值 |
|------|------|
| UI 深度 | 子 + 孙（`max_depth=2`） |
| 门禁深度 | **全部后代**（任意深度；错误 payload 列出未关闭者，便于处理更深节点） |
| 终态判定 | 与 `taskEvents.ResolveTerminalKind` 一致：列名 ∈ {已完成,completed / 已取消,cancelled,canceled}，或 `completed=true` |
| 门禁触发 | 目标状态为终态，且与当前状态不同（进入终态） |
| 无子树 | 门禁直接通过 |
| 错误码 | HTTP 409，`code=DESCENDANTS_NOT_TERMINAL` |
| 列表 API | `GET /api/tenant/{tid}/workspace/{wid}/todos/{id}/subtree/?max_depth=2` |

## 4. API 契约

### 4.1 GET subtree（展示）

```
GET /api/tenant/{tenant_id}/workspace/{workspace_id}/todos/{task_id}/subtree/?max_depth=2
```

响应 200：

```json
{
  "task_id": "...",
  "max_depth": 2,
  "summary": {
    "total": 3,
    "settled": 1,
    "open": 2,
    "completed": 1,
    "cancelled": 0
  },
  "nodes": [
    {
      "id": "child1",
      "title": "...",
      "parent_task": "<root>",
      "depth": 1,
      "progress_column_id": "...",
      "progress_column_name": "进行中",
      "completed": false,
      "terminal_kind": "",
      "settled": false
    },
    {
      "id": "gc1",
      "title": "...",
      "parent_task": "child1",
      "depth": 2,
      "progress_column_id": "...",
      "progress_column_name": "已完成",
      "completed": true,
      "terminal_kind": "completed",
      "settled": true
    }
  ]
}
```

权限：与 GET 任务相同（workspace 可读成员）。

列名解析：TTS 一次拉取工作区 progress-system columns（既有 Django/转发路径），建 id→name 映射；映射失败时 `progress_column_name` 可空，仍可用 `completed` 判定 settled。

### 4.2 PATCH / switch 门禁

当 `ResolveTerminalKind(newColumnName, completedBecameTrue) != ""` 时：

1. 加载全部后代（BFS/`parent_task_id` 反查）。
2. 解析各后代列名；若任一 `!IsTerminalState(...)` → **409**：

```json
{
  "error": "存在未完成或未取消的子任务，无法关闭当前任务",
  "detail": "存在未完成或未取消的子任务，无法关闭当前任务",
  "code": "DESCENDANTS_NOT_TERMINAL",
  "open_descendants": [
    {"id": "...", "title": "...", "depth": 1, "progress_column_name": "进行中"}
  ]
}
```

3. 全部 settled → 照常更新并发布 `TASK_STATUS_CHANGED`。

`/switch` 在 `completed` 变为 true 时同样走门禁。

## 5. 领域概念（轻量 → Step 6）

| 概念 | 说明 |
|------|------|
| **TaskSubtree** | 以任务为根、按 `parent_task_id` 展开的有界子树视图 |
| **TerminalKind** | `completed` \| `cancelled` \| "" |
| **DescendantTerminalGate** | 领域服务：进入终态前校验全部后代 settled |
| **SubtreeSummary** | total / settled / open / completed / cancelled 计数 |

## 6. UI

在 `TaskDetailTaskIdentityPanel` 下方（或身份区底部）新增 **「下级交付物」** 面板：

- `data-testid="task-subtree-status"`
- 摘要：`已关闭 n/m`
- 列表：depth=1 缩进 0；depth=2 缩进一级；展示标题 + 进度列徽章；可点击跳转子任务详情
- 无子树：不渲染面板（或 `v-if="summary.total > 0"`）
- 进度变更 409：`progressStatusError` 展示服务端文案 + `data-traceId`

工作面板看板拖拽若走同一 PATCH，自动获得门禁；错误提示沿用既有 toast/错误展示规范。

## 7. 🏛️ 架构变更影响

- **有**：扩展 TTS 读路径（subtree）与写路径门禁；Vue 详情消费新 API
- 架构文件：`v41-application-integration-20260719-1816-claude.{puml,archimate,mermaid.md}`
- Python 新增接口：无

## 8. 改动文件清单（预期）

| 文件 | 变更 |
|------|------|
| `taskTaskService/src/subtree_*.go` | subtree 查询 + 终态工具 |
| `taskTaskService/src/terminal_gate.go` | DescendantTerminalGate |
| `taskTaskService/src/task_handlers.go` / `route_handlers.go` | 接线 GET/门禁 |
| `taskTaskService/src/*_test.go` | 单测 |
| `TaskDetailSubtreeStatusPanel.vue`（新） | UI |
| `taskDetailFetchFns.js` / `useTaskDetail.js` | 拉取 + 进度错误解析 |
| `docs/intents/...` | 意图 + 测试意图 |
| `docs/architecture/v41-*` + VERSION_HISTORY | 架构 |

## 9. 业务意图 → 事件

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外 |
|----------|--------|--------|--------|------|
| 查看子树状态 | — | — | — | 纯查询 |
| 父任务进入终态（门禁通过） | `TASK_STATUS_CHANGED`（既有） | TTS update/switch | taskEvents 释放机器 / work-panel SSE | — |
| 门禁拒绝 | — | — | — | 无状态变更，不发事件 |

## 10. 验收标准（汇总）

1. 有子/孙任务时详情可见状态与完成摘要。
2. 存在未关闭后代时无法将父任务标为已完成/已取消（API 409）。
3. 全部后代已关闭后可正常进入终态并触发既有 `TASK_STATUS_CHANGED`。
4. 错误 UI 带 `data-traceId`（有响应头时）。
