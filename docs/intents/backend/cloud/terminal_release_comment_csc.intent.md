# 意图：任务终态按评论 CSC 释放运行资源

## 背景与目标

v80 后任务级 CSC 不再持有 `instance_id` / `server_url`。`TASK_STATUS_CHANGED` 的 `release-servers-on-terminal` 仍把空模板行当成「资源未就绪」重试，11 次后 DLT（例：`trace_id=61544b00-9f30-4c30-97d6-c57264e2cda7`）。

目标：终态只释放该任务下**已启动的评论 CSC**；无运行资源时 success no-op，不再 DLT。

## 范围与边界

- 范围内：`list-by-task` internal API；`taskstatuschanged` handler 按评论释放；`CLOUD_SERVER_STOPPED` 带 `comment_id` + 自洽 `instance_id`/`region_id`。
- 范围外：bootstrap `no task-level start event payload`；其他 `LoadForTask` 消费者（graceful-await / started / stopped 回退）；Starting 晚到 VM 的补偿。

## 约束与风险

- 禁止把 `comment_id=''` 当运行实例。
- 释放谓词：**有 `instance_id` 且未处于已停机终态**（Running / Starting / Pending / Initializing / 空状态 / mock）均须释放；另处理非空 `server_url`。
- 与工作面板「已启动」计数谓词 `machineRuntimeCountsAsStarted` **刻意分离**（计数不含 Starting；终态释放必须含 Starting+instance）。
- Starting **无** instance / 计数 0 / 无评论运行态 → success，禁止 retry 耗尽。
- 不新增 Python 接口；不新增事件类型。
- Kafka 一次性消费失败（cloud 不可达 → DLT）不得让已取消/已完成任务的机器继续跑：cloud 入站 heartbeat 等路径须查终态并同步释放；`cloud_csc_reconcile` 须补偿无心跳的运行 CSC。查找失败 fail-open。

## 验收标准

1. 空模板 + 无评论运行态 → `DispatchSuccess`，无停机事件。
2. 一评论 Running 有 instance → 一条 `CLOUD_SERVER_STOPPED`（含 `comment_id`）。
3. 两评论均已启动 → 两条停机事件，key 含 comment。
4. Starting 且无 instance → success no-op。
5. Starting **且有** instance → 一条 `CLOUD_SERVER_STOPPED`（进度切到已完成/已取消时不漏杀开机中的 VM）。
6. 本 DLT 信封三元组复现通过。
7. 任务已取消但 `TASK_STATUS_CHANGED` 已 DLT：下一次 heartbeat 返回 410 `TASK_TERMINAL` 且 CSC `terminal_released=1`、`instance_id` 清空。
8. 进行中任务 heartbeat 不被 410。
9. `reconcile-orphan-csc` 同 tick 释放已取消但仍有 instance 的评论 CSC。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 任务进入已完成/已取消 | TaskStatusChanged | Kafka `task-status-changed` | taskTaskService（已有） | release-servers；fanout-sse | — |
| 终态释放一台评论云主机 | CloudServerStopped | Kafka `cloud-server-stopped` | release-servers | 删实例 + clear-after-stop | — |
| 终态无运行中评论资源 | — | — | — | success no-op | 无状态变更 |
| 查询任务下评论 CSC | — | — | list-by-task | — | 纯查询 |
| 入站发现任务已终态 | CloudServerStopped | Kafka `cloud-server-stopped` | `releaseMachineForTerminal`（heartbeat 410 补偿） | 删实例 + clear-after-stop | 与 release-servers 同一事件；Kafka 空则仅打标 |
| 对账发现任务已终态 | CloudServerStopped | Kafka `cloud-server-stopped` | `reconcileTerminalTaskCSCs` | 同上 | 同上 |
| 批量查任务终态 | — | — | `POST /api/internal/tasks/terminal-kinds/` | cloud 守卫/对账 | 纯查询 |

## 实施计划

1. taskCloudService `list-by-task` + Swagger。
2. taskEvents 客户端 + handler 改按评论释放。
3. 单测覆盖 S1–S5。
4. 入站守卫 + 终态对账补偿 DLT/cloud 宕机窗口。

## 变更记录

- 2026-08-21：补偿 `TASK_STATUS_CHANGED` DLT / cloud 不可达：heartbeat 410 + `reconcileTerminalTaskCSCs`。
- 2026-08-14：由 DLT 立项；范围 = 评论级逐台释放 + 无资源 no-op。
