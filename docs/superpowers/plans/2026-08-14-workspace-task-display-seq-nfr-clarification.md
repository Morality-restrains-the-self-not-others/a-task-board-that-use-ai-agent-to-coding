# NFR Clarification: 工作空间任务帖人读序号

> Design: `docs/superpowers/specs/2026-08-14-workspace-task-display-seq-design.md`
> Value stream: `docs/superpowers/plans/2026-08-14-workspace-task-display-seq-value-stream.md`

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 级别 | 动作 |
|------|---------|----------|------|------|
| POST 创建任务 | tenant + workspace | tenant 合适 | L2 | 发号键必须含二者 |
| GET 列表/详情 | tenant + workspace (+ task id) | tenant 合适 | L2 | — |
| 搜索 `#N` / `N` | tenant + workspace **必须** | tenant 合适；seq 不是分片键 | L2 | **禁止**无 workspace 按 seq 全局查 |
| `TASK_CREATED` | tenant + workspace + task_id | tenant 合适 | L2 | payload 增补 seq，路由键不变 |
| 发号 SQL | tenant + workspace PK | 一行/工作空间 | L0 | 小表，不分片；升级触发：单库工作空间数 > 1e6 |

## 支撑程度（默认 L2）

| 类别 | 级别 | 说明 |
|------|------|------|
| 一致性 | L2 | 发号与插帖同事务；失败回滚不漏号或空洞可接受（回滚则不提交） |
| 性能 | L2 | 行锁一行发号器；创建 QPS 远低于 mill |
| 安全 | L2 | 搜索强制工作空间；客户端不可指定 seq |
| 可伸缩性 | L2 | 分片键 tenant；seq 仅工作空间内唯一 |
| 可观测性 | L2 | `event=workspace_seq_allocated task_id=… seq=N` |
| 可用性 | L1 | 发号失败则创建失败（正确优先） |

## 质量场景

1. **刺激**：同一工作空间并发两创建。**响应**：两个不同 seq，无唯一键冲突。
2. **刺激**：搜索 `#2` 且他空间也有 seq=2。**响应**：只命中当前空间。
3. **刺激**：删帖后再建。**响应**：序号不复用已分配值。

## 领域模型影响

- `WorkspaceSeq` 是 Task 上的值对象（正整数），不是聚合根。
- `WorkspaceTaskSeq` 是同上下文发号设施，与 Task 同事务，不独立发布事件。
