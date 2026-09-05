# 意图：评论容器启动 TraceId 按评论持久化

## 背景与目标

任务详情「启动 TraceId」原先依赖 SSE 内存或调度日志后缀，且 `bindStartVmTraceContext` 把 run trace 设成 `task_id`。同一任务多条评论各自启容器时会撞上同一个 ID，冷打开后行消失。

目标：每次 start-vm **独立** TraceId（禁止 `task_id`）；写入该评论 binding 的 `start_trace_id`；list API 回传；HTTP 200 带 `trace_id`。

## 范围与边界

- 范围内：`cloud_comment_container_bindings.start_trace_id`；`bindStartVmTraceContext`；start-vm / start-vm-auto / 同评论 inflight 附着 / heal 绑 instance；list JSON；无 `comment_id` 时不扇出同一 trace。
- 范围外：不新增服务/Kafka 事件；不查 Loki。日志后缀仍可 extract。

## 约束与风险

- 入站 `X-Trace-Id` 合法且不等于 `task_id` 时保留；否则 `tracelog.NewTraceID()`。
- 有 `comment_id` 只写该行；无 `comment_id` 不写任何 binding 的 `start_trace_id`，扇出日志不得附带同一 `trace_id=`。
- DDL 走 `dataMigrate/taskCloudService/018_*` + 9999，业务进程不 migrate。

## 验收标准

1. 同任务两次 `bindStartVmTraceContext` 得到两个非空、互不相同、且都不等于 `task_id` 的 ID。
2. 两评论 persist 后 list JSON 各自 `start_trace_id` 不同。
3. persist(`task_id`) 被拒绝，列保持原值或空。
4. start-vm-auto 同评论 inflight 附着 200 体含非空 `trace_id` 且不等于 `task_id`。
5. start-vm persist 早于 binding INSERT（第二条 @镜像 常见竞态）时，ensure/create 后 list 仍有该评论独立 `start_trace_id`。
6. 同任务仅有第一条 comment-scoped start 事件、无 `comment_id=''` 任务级行时，第二条评论 `loadStartEventPayload` 仍能取到镜像/网络参数。
7. heal/`persistStartVmInstanceBinding` 成功且列为空时写入独立 `start_trace_id`（≠ `task_id`）；已有独立 ID 不覆盖。
8. list 时 binding 已有 `csc_id` 但 `start_trace_id` 为空，补写独立 ID 并打 `binding_start_trace_id_ensured`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 记录本评论容器启动 TraceId | — | — | start-vm 写 binding 列 | — | 同一次启机的附属持久化，不新增领域事实；启机事件仍为既有 `CLOUD_SERVER_STARTED` / SSE |

## 变更记录

- 2026-08-16：第二条 @镜像 缺 TraceId——bootstrap 找不到任务级 start 载荷、heal 绑 instance 不写列。改为 sibling start 载荷回退 + instance bind/list 补写独立 ID。
- 2026-08-16：persist 早于 binding 行时暂存，INSERT/ensure 后 drain 回写（第二条评论缺 TraceId）。
- 2026-08-13：禁止 task_id；同任务多评论独立 TraceId；binding 一等字段持久化。
