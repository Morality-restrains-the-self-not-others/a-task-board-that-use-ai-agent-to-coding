# DDD：任务子树状态与终态门禁

## 限界上下文

任务协作（taskTaskService）

## 领域模型

| 类型 | 名称 | 职责 |
|------|------|------|
| Entity | Task | `parent_task_id`、`progress_column_id`、`completed` |
| Value Object | TerminalKind | completed / cancelled / empty |
| Value Object | SubtreeSummary | total/settled/open/completed/cancelled |
| Domain Service | DescendantTerminalGate | 进入终态前校验后代 |
| Read Model | TaskSubtreeView | depth 有界展示节点列表 |
| Domain Event | TASK_STATUS_CHANGED | 门禁通过后既有发布 |

## 端口

- `ProgressColumnCatalog`（出站）：workspace → id→name 映射
- `TaskRepository`：按 parent 列出子节点 / 加载任务

## 架构影响

见 `docs/architecture/v41-*`；无新聚合根。
