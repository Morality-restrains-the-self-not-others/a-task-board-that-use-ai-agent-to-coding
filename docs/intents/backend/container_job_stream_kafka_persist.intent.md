# 意图：容器执行步骤经 Kafka 再 SSE，消费者落库，前端初始化先拉历史

## 背景与目标

容器 agent 执行步骤此前由 taskContainerGateway **轮询** `GET /jobs/:id/events` 后直接打 Kafka `SSE_MESSAGE`；前端另用 2s `setInterval` 打 `container-job-execution-log`（代理到容器内存）。SaaS 库无步骤存档，页面刷新只能等容器在线。

目标链路：

1. 容器 **PUSH** `job-stream-push` 到 taskCloudService
2. Cloud **先发布** Kafka `SSE_MESSAGE`（`status=container_job_stream`），再由既有 `1_send_sse_message` 经 Redis/SSE 到前端
3. 新消费者 `2_persist_job_execution_event` 把步骤/生命周期写入 `cloud_job_execution_event`
4. 前端任务详情初始化：**先 GET 数据库历史**，再依赖 SSE 增量；禁止无触发后台轮询

## 范围与边界

- 范围内：onlineServiceJS `recordJobEvent` 出站；Cloud inbound + GET hydrate；taskEvents 第二 intent；停用网关自动 poll 与前端 active-job poller
- 范围外：不改层图/心跳协议；不把 chunk 文本落入数据库（仅 SSE 直播，避免热表膨胀）

## 约束与风险

- 业务进程禁止 ticker 扫容器；周期工作不走 Gateway poll（ADR-0011）
- 表前缀 `cloud_`、utf8mb4、按 `created_at` RANGE 分区；幂等 `(task_id, job_id, seq)`
- 落库只经 Kafka 消费者调 Cloud internal API，inbound handler **不写库**
- Kafka 发布失败须让容器重试（HTTP 502），避免「ACK 了但前端永远看不到」

## 验收标准

1. `POST .../server-container-token/job-stream-push/` 成功路径发布 `SSE_MESSAGE`，响应不依赖 DB 写入
2. 消费者对 `phase=step|start|running|completed|failed|interrupted` 落库；`chunk` 不落库
3. `GET compute/container-job-execution-log` 从 DB 还原 `{job, steps}`，无需容器在线
4. 前端选中 job 时即使 `containerEndpointRegistered=false` 也会 GET 历史；不再 `setInterval` 拉执行日志
5. Gateway 默认不再因 `container-layer-command` 启动 events 轮询

## 实施计划

1. `dataMigrate/taskCloudService/023_cloud_job_execution_event.sql`
2. Cloud `handleJobStreamPush` + GET hydrate + internal persist
3. taskEvents `sse_message/2_persist_job_execution_event`（port 18062）
4. 容器 `publishJobStreamEventToSaas` 挂到 `recordJobEvent`
5. 停用网关自动 poll、前端 poller；补意图测试

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---|---|---|---|---|---|
| 容器上报执行步骤/作业生命周期 | SSE_MESSAGE | Kafka topic `sse-message`；`data.status_data.status=container_job_stream` | taskCloudService `handleJobStreamPush` → `publishTaskSSE` | `1_send_sse_message` → Redis `sse:{task_id}`；`2_persist_job_execution_event` → `cloud_job_execution_event` | — |
| 页面拉取历史执行步骤 | — | — | GET `container-job-execution-log` | 读库 | 纯查询，无状态变更 |
