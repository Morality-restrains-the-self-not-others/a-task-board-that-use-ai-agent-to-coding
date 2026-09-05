# 意图：任务级只标记运行中机器数与容器数

## 背景与目标

任务级 CSC（`comment_id=''`）仍被当成「这台任务的服务器」写入 `instance_id` / `last_runtime_status`，与评论级运行态冲突。

目标：任务级只标记该任务有多少台评论机器在运行、多少个评论容器在运行；生命周期只存在评论 CSC。

## 范围与边界

- 范围内：`running_machine_count`、`running_container_count`；评论行 runtime/URL 变更后重算；indicators / 任务壳读两计数；禁止任务级写运行态。
- 范围外：评论 CSC 模板克隆仍从任务级行读 platform/region/auth；不删除任务级行。
- **消费侧缺口（2026-08-14）**：`TASK_STATUS_CHANGED` 终态释放已另开意图 `backend/cloud/terminal_release_comment_csc`（按评论 CSC 释放，禁止再读任务级 instance）。

## 约束与风险

- 机器运行中 = `machineRuntimeCountsAsStarted`；Starting 不计入。
- 容器运行中 = `containerReachabilityCountsAsRunning`（已启动且 `server_url` 非空）。
- 不得 `UPDATE last_runtime_status WHERE task_id=?` 全刷。
- 无 comment 的 start persist 不得写任务级 instance。
- **不兼容存量任务**：DDL 直接清空任务级 `instance_id` / `last_runtime_status` / `public_ip` / `server_url`；禁止 `healCommentCSCInstanceFromTaskLevel` 从任务级领养 instance。仅有任务级 instance、评论行仍空的任务视为未启动。

## 验收标准

1. 一评论 Running 无 URL → 机器 1、容器 0；补上 server_url → 皆 1。
2. 两评论各 Running 且可达 → 机器 2、容器 2。
3. 仅 Starting → 两计数均为 0。
4. Released → 两计数减对应评论。
5. runtime-status 无 comment_id → 400。
6. indicators 忽略模板行 instance；`machine_running`/`container_running` 分别随两计数。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 评论机器/容器进入或离开运行中并更新任务计数 | TaskRunningCountsRecomputed | — | `recomputeTaskRunningCounts` | 模板行两列更新 | 证据豁免：同服务同库投影，不跨边界 |
| 查询运行中数量 | — | — | indicators / 详情只读 | — | 纯查询，无对应事件 |

## 实施计划

1. DDL 加两列，并清空任务级运行态字段；两计数只按评论行重算。
2. 评论 runtime/URL 写路径后重算。
3. 收窄 persist / last_runtime_status / snapshot；删除任务级→评论 heal。
4. FE 看板与任务壳展示 N/M。

## 变更记录

- 2026-08-14：由「任务级仍存 Starting/Running」改为「只标记运行中数量」。
- 2026-08-14：补充必须同时记录运行中机器数与运行中容器数。
- 2026-08-14：可清空旧数据，不兼容存量任务；删除 heal 路径。
