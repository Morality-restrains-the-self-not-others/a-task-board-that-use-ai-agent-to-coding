# DDD：Fork 确认弹窗副本数量

- **日期**: 2026-08-23
- **价值流**: `docs/superpowers/plans/2026-08-23-fork-copy-count-value-stream.md`
- **NFR**: `docs/superpowers/plans/2026-08-23-fork-copy-count-nfr-clarification.md`

## 限界上下文

`Task` 上下文（taskTaskService + taskFE 任务详情）。本增量不跨上下文。

## 值对象

`ForkCopyCount`：整数，不变量 `1 <= n <= 99`。非法输入 clamp，不抛给用户技术错误。落点 `taskFE/app/src/utils/forkCopyCount.js`。

## 聚合

沿用 `Task`。每份副本是独立聚合实例（独立 Snowflake id、独立 `fork_from` 指向源任务）。不引入 `ForkBatch` 聚合（前端 batch key 仅用于幂等，不落库）。

## 领域事件

| 业务意图 | 事件 | 发布点 | 例外 |
|----------|------|--------|------|
| 派生 N 个副本 | `TASK_CREATED` × N | 既有 `handleCreateTask` → publish | 不新增 `TASKS_FORKED_IN_BATCH`；每份独立消费（SSE 看板、配额、auto_run） |

## 领域服务

无新后端领域服务。前端 `forkTask` 是应用服务：校验数量、顺序创建、部分失败策略。

## 端口

不新增 Repository / EventBus 端口。HTTP 适配器仍为既有 todos POST。
