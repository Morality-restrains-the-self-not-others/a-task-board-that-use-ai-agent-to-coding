# NFR 澄清：Fork 自动运行按智能体（模型）复制

- **日期**: 2026-08-25
- **默认等级**: L2 Standard；创建写路径幂等 L3（配额/启服）
- **价值流**: `docs/superpowers/plans/2026-08-25-fork-auto-run-agent-copies-value-stream.md`

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 可伸缩性 | 动作 |
|------|---------|----------|----------|------|
| 任务详情 `/tenant/:tenantId/workspace/:workspaceId/task-detail/:taskId/` | tenantId + workspaceId + taskId | 是，与任务表租户/工作空间/任务对齐 | L0 本增量不改路由 | 无 |
| `GET /api/cloud/feature-params/tenant_id/{tid}/…` | tenantId（+ workspaceId） | 是 | L0 打开弹窗拉一次，禁止轮询 | 无 |
| `GET /api/personal/feature-params-configs/` | 当前用户（会话） | 是，个人配置按 user | L0 | 无 |
| `POST /api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/` × N | tenantId + workspaceId | 是 | L0 理由：单次点击最多 99 次顺序 POST；升级触发：产品需要 >99 或异构批接口时再评估 | 无 |
| 内部 `POST …/container-agent-comments` | tenantId + workspaceId + taskId（body） | 是 | L0 每份一次 | 无 |
| 新标签 task-detail（第一份副本） | 同上 | 是 | L0 | 无 |
| `TASK_CREATED` × N | 事件体含 tenant/workspace/task | 是 | L0 沿用既有发布 | 无 |

Hard Gate：全部路径已携带合适分片键或已书面 L0。可伸缩性类别纳入第 5 步为 L0。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| 确认派生 → POST × N | 写 N 条任务；每条扣配额、发 `TASK_CREATED`、auto_run 启服 | 双击确认、超时重试、循环内重试单份 | 一次点击的第 i 个所选模型副本 | 点击生成 `batchKey`；第 i 份 `Idempotency-Key=${batchKey}:${i}`（i 与勾选模型顺序同粒度） | 同键返回已创建任务，不二次扣费/启服；不同 `${batch}:${i}` 创建不同任务 |
| 智能体资源/模型勾选 | 无 | — | — | L0 只读 UI | — |
| 取消 / 关遮罩 | 无 | — | — | L0 | — |
| pending agent 创建 | 写 AIComment 行 | auto_run 重入 `ensureAutoRunAtComment` | 该任务当前 active auto_run agent | 既有 reuse-by-task | 复用已有 comment，不双建 |
| 容器首 job | 创建 trae job + 覆盖模型 env | bootstrap 重放 | 该容器 auto_run 首指令 | 既有 `auto_run_first_job.json` marker | 有 marker 跳过 |
| `TASK_CREATED` 消费 | 看板 SSE 等 | Kafka 重投 | 单任务 id | 既有消费者幂等（非本增量改键） | 禁止用 tenant_id 作消费键 |

资金/配额：每份走既有 `consumeTaskPostQuota`。auto_run 为云资源路径，单份已 ≥ L3；批量不合并为一笔资金事务。禁止用 `company_id`/`tenant_id`/`user_id` 作本增量写路径键。

前端防重放：`createClickGuard` 同步门闩 + 进行中 disabled/aria-busy；同一次意图回传同一 `batchKey`；禁止每次 HTTP 新建 UUID。

## 适用 NFR 类别与等级

| 类别 | 等级 | 说明 |
|------|------|------|
| 可伸缩性 | L0 | 路径已带租户/工作空间；单次 ≤99 顺序 POST |
| 数据一致性 | L3 | 配额/启服按份幂等 |
| 容错机制 | L2 | 部分失败保留已创建副本并打开第一份 |
| 性能 | L2 | 打开弹窗拉配置一次；禁止 setInterval |
| 安全 | L2 | 个人配置 IDOR 既有检查；模型名属配置清单 |
| 可观测性 | L2 | 结构化 info：copy_index、model（无密钥） |

## 质量场景

- 刺激：打开弹窗。响应：无数量框；未选派生方式时确认 disabled。
- 刺激：仅派生确认。响应：1 次 POST，`auto_run=false`，无 `agent_models`，无 `fork_count`。
- 刺激：自动运行勾选两模型。响应：2 次 POST，`agent_models[0].model` 不同，键 `${batch}:1` 与 `:2`。
- 刺激：确认连点。响应：仅一次 batch。
- 刺激：第 2 份 402。响应：保留第 1 份并打开，提示部分失败。
- 刺激：切换智能体资源。响应：已选模型清空并预勾新默认。

## 领域模型影响

值对象 `AgentModelRef{provider, model}` 挂在本次 auto_run 评论/job context，不落任务新列。聚合仍为每份独立 `Task`。事件仍为 `TASK_CREATED` × N。幂等键粒度 = 第 i 个所选模型，与 NFR 表一致。
