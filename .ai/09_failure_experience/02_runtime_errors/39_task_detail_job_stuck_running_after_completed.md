# [运行时] 任务详情层图指令已完成仍显示 running

## 现象

任务详情 zTree 可写层/指令行标题以 `running …` 开头，title 为 `状态: running`，但容器内该 job 已是 `completed`（`GET /api/layers` 的 `job_status` / `mind_state` 已终态）。

典型路径：长日志 Trae 任务结束后刷新或保持打开详情页。

## 环境与上下文

- 展示：`layerZtreeNodes` 合并层行时用 `job.status`（`topicForLayerMergedWithJob`）
- 推送：`taskContainerGateway` job-stream 轮询 `GET …/jobs/:id/events` 与 `GET …/jobs/:id`，终态发 SSE `container_job_stream`（`phase=done`）
- 前端：`updateServerStatus` 对 `container_job_stream` 在终态时 `refreshLayerGraphFromServer(true)`

## 根因

1. **旧 onlineServiceJS 镜像** 的 `GET /api/jobs/:id` 仍内嵌全量 `output`（可达十余 MB），网关轮询超时，**发不出** `phase=done`
2. events 已含 `phase=completed`，但前端原先**只**在 `done`/`error` 刷层图，忽略 `completed`/`failed`/`interrupted`
3. 未识别的 job-stream phase 还会落入启动状态分支，干扰其它 UI
4. **（2026-07-19）双源冲突**：执行日志 `meta=0` stub 带 `status:""`，前端 `{...jobHint, ...stub}` 把层图 `running` 盖成空串 → 代理步骤徽章凭步骤态显示「已结束」；zTree 仍读层图 `jobs[].status=running`。刷新层图时滞后 `/api/jobs` 还会把本地终态盖回 `running`

## 修复

- `taskContainerGateway/src/job_stream.go`：events 出现终态 phase 时立即发 `done` 并结束轮询（不依赖巨量 job JSON）
- `updateServerStatus.js`：终态含 `completed`/`failed`/`interrupted`；乐观 `patchLayerGraphJobStatus`；未知 phase 直接 return
- `taskContainerGateway` `handleContainerLayerGraph`：`GET /api/jobs` 失败时降级为空 `jobs`，仍返回 `layers`（含 `job_status`）；Django `container-layer-graph` / `container-job-execution-log` 已始终 410，无 forward fallback
- `l0_job_execution_log.go`：`meta=0` stub **省略** `status` 字段（禁止空串）
- `fetchJobExecutionLogBySteps.js`：`mergeJobMetaPreserveHint` + 步骤终态推断 `inferJobStatusFromAgentSteps`
- `taskDetailExecLog.js`：执行日志确认终态后 `patchLayerGraphJobStatus` 回写层图
- `taskDetailLayerGraphPayload.js`：`reconcileLayerGraphJobs`（层 `job_status` 终态优先；禁止同 job 终态回退为 active）

## 预防

- 容器镜像须带 `jobToApiDict` 默认省略 `output`（`include_output=1` 才全文）；见 `OPT-20260718-025`
- job-stream 终态以 **events** 为准，GET job 仅作兜底
- 新增 SSE phase 时前端须显式处理或安全 return，禁止落入启动状态默认分支
- 层图转发不得因 jobs 超时整单失败
- **执行日志**按 `GET …/steps?after_step&limit` 分页拉取；Gateway/`container-job-execution-log` 优先 steps，`meta=0` 跳过巨量 job JSON
- `meta=0` stub **不得**写空 `status`；层图展示与代理步骤徽章须同源或终态回写一致
