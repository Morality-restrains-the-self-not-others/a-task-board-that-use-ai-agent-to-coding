# 测试意图：评论执行细节冷打开展示独立启动 TraceId

## 测试目标

验证前端按评论回填 `start_trace_id`，不用 `task_id` 冒充。

## 测试分层

- 单元：`useCommentContainerBindings.test.js`、`bindingStartTraceIdFromLogs.test.js`、`startVmHttpResult.test.js`

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | list `start_trace_id=cmt-start-aaa`，日志另有 `trace_id=task_cold_open_1` | refresh | `bindingStartTraceIdFor`=`cmt-start-aaa` |
| T2 | 无列值、无日志，`statusTraceId` 为任务 ID 形态 | buildPerBinding | `startTraceId===''` |
| T3 | 日志 `trace_id=` 等于当前 taskId | refresh | 不回填 |
| T4 | HTTP body `trace_id` 与 `_traceId` 不同 | buildStartVmAcceptedStatusUpdate | 采用 body `trace_id` |
| T5 | SSE `comment_id`+`trace_id` 且 message 为空 | updateServerStatus | `latestServerStartupStatusForBinding.trace_id` 仍写入该评论 |

## 自动化落点

- `taskFE/app/src/composables/taskDetail/useCommentContainerBindings.test.js`
- `taskFE/app/src/utils/bindingStartTraceIdFromLogs.test.js`
- `taskFE/app/src/utils/startVmHttpResult.test.js`
