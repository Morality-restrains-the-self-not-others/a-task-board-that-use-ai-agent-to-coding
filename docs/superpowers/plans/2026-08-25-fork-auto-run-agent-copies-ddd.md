# DDD：Fork 自动运行按智能体（模型）复制

- **日期**: 2026-08-25
- **价值流**: `docs/superpowers/plans/2026-08-25-fork-auto-run-agent-copies-value-stream.md`
- **NFR**: `docs/superpowers/plans/2026-08-25-fork-auto-run-agent-copies-nfr-clarification.md`

## 限界上下文

| 上下文 | 服务 | 本增量职责 |
|--------|------|------------|
| Task | taskTaskService + taskFE | 派生创建；每份独立聚合；可选 `agent_models` 传入 auto_run |
| FeatureParams | taskCloudService | 只读：配置与 `supported_models`；不改 schema |
| AutoRun / AIComment | taskTaskService + taskAIComment + trae-agent | pending agent `context_pack.agent_models`；首 job env 覆盖 |

不新拆上下文。任务默认模型仍由 `feature_params_source` 解析；本次运行模型是 job 级值对象。

## 值对象

- `AgentModelRef`：`provider` + `model` 均非空。auto_run 创建请求至多 1 项。
- `ForkCopyIndex`：与勾选模型顺序 1..N 对齐，构成幂等键后缀。
- 沿用 `ForkCopyCount` clamp（1–99）仅作为 N = 勾选数的上限，不再作为仅派生输入。

## 聚合

沿用 `Task`。每份副本独立 Snowflake id、独立 `fork_from`。不引入 `ForkBatch` 聚合（前端 batch key 仅幂等，不落库）。模型绑定在 auto_run 评论/agent context，不写入任务行（NFR：本迭代不改 schema）。

## 领域事件

| 业务意图 | 事件 | 发布点 | 例外 |
|----------|------|--------|------|
| Fork 仅派生 1 份 | `TASK_CREATED` | 既有 `createTaskOnce` → publish | 不新增事件 |
| Fork 自动运行按模型复制 | `TASK_CREATED` × N | 每份 `createTaskOnce` | 不新增 `TASKS_FORKED_WITH_MODELS`；差异在 job context |

## 领域服务

- 前端 `forkTask`：仅派生恒 1；自动运行按模型顺序创建、部分失败策略。
- 后端 `parseCreateTaskAgentModels`：auto_run 时可选字段校验。
- 容器 `maybeStartAutoRunFirstInstruction`：从 context_pack 取 `AgentModelRef` 写入 job `env`。

## 端口

不新增 Repository / EventBus。HTTP 仍为既有 todos POST 与 internal pending-agent。EventBus 端口沿用既有 `publishTaskCreated`。
