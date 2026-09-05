# NFR 澄清：Fork 确认弹窗副本数量

- **日期**: 2026-08-23
- **默认等级**: L2 Standard；创建写路径幂等 L3（配额/可能启服）

## 路径分片键审视

| 路径 | 分片 ID | 是否合适 | 可伸缩性 | 动作 |
|------|---------|----------|----------|------|
| 任务详情 `/tenant/:tenantId/workspace/:workspaceId/task-detail/:taskId/` | tenantId + workspaceId + taskId | 是，与任务表租户/工作空间/任务对齐 | L0 本增量不改路由 | 无 |
| `POST /api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/` × N | tenantId + workspaceId | 是 | L0 理由：单次点击最多 99 次顺序 POST，不引入无键列表；升级触发：产品需要 >99 或后端批量接口时再评估 | 无 |
| 新标签 task-detail（第一份副本） | 同上 | 是 | L0 | 无 |

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| 确认派生点击 → POST × N | 写 N 条任务；每条可能扣配额、发 `TASK_CREATED`、auto_run 启服 | 双击确认、超时重试、同 batch 循环内重试单份 | 一次点击要创建的第 i 份副本（i=1..N） | 点击生成 `batchKey`；第 i 份 `Idempotency-Key=${batchKey}:${i}` | 同键返回已创建任务，不二次扣费/启服；不同 `${batch}:${i}` 创建不同任务 |
| 数量输入 | 无 | — | — | L0 只读 UI | — |
| 取消 / 关遮罩 | 无 | — | — | L0 | — |

资金/配额：每份走既有 `consumeTaskPostQuota`。auto_run 为云资源路径，单份已 ≥ L3；批量不合并为一笔资金事务。

前端防重放：`createClickGuard` 同步门闩 + 进行中 disabled/aria-busy；禁止每次 HTTP 新建 batch UUID。

## 质量场景

- 刺激：打开弹窗。响应：数量为 1。
- 刺激：输入 0 / 100 / 空后失焦或确认。响应：clamp 到 1 或 99。
- 刺激：数量 3 点「仅派生」。响应：3 次 POST、3 个不同 Idempotency-Key 同 batch 前缀、只 open 第一份。
- 刺激：确认连点。响应：仅一次 batch。
- 刺激：第 2 份 402 配额不足。响应：保留第 1 份并打开，提示部分失败。

## 领域模型影响

新增前端值对象 `ForkCopyCount`（1–99）。不新聚合。不新领域事件：N 次既有 `TASK_CREATED`。
