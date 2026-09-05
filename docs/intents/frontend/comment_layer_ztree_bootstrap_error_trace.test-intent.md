# 测试意图：任务关联引导克隆失败须带 data-traceId

## 测试目标

验证 `container_bootstrap_failed` SSE 的 `trace_id` 写入独立 ref，并挂到 ztree 错误 DOM 的 `data-traceId`；无值则省略。

## 测试分层

| 层 | 落点 |
|----|------|
| 单元 | `taskCloudService/src/container_runtime_event_test.go` |
| 单元 | `taskFE/app/src/composables/taskDetail/containerBootstrapSse.test.js` |
| 单元 | `taskFE/app/src/composables/taskDetail/updateServerStatus.test.js` |
| 单元 | `taskFE/app/src/composables/taskDetail/taskDetailCommentsSectionHelpers.test.js` |
| 组件 | `taskFE/app/src/components/task-detail/TaskDetailCommentLayerZtreeStatus.test.js` |

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | runtime-event body 含 `trace_id` | `firstNonEmptyTraceID` | 返回该值（优先于 fields） |
| T2 | SSE `container_bootstrap_failed` + `trace_id` | `updateServerStatus` | `containerBootstrapFailureTraceId` 有值；`statusTraceId` 仍空 |
| T3 | SSE 无 `trace_id` | `updateServerStatus` | bootstrap trace ref 为空，不写 unknown |
| T4 | 引导失败文案 + trace | `resolvePerCommentLayerZtreeUi` | `commentLayerZtreeLoadingErrorTraceId` 等于 SSE trace |
| T5 | `containerLayerGraphAuthInvalid` | `resolveCommentLayerZtreeLoadingErrorTraceId` | 空（不挂引导 trace） |
| T6 | 错误态 + trace | 挂载 ZtreeStatus | `[data-testid=comment-layer-ztree-loading-error]` 有 `data-traceId` |
| T7 | 错误态无 trace | 挂载 | 错误节点无 `data-traceId` |
| T8 | `container_bootstrap_complete` | `updateServerStatus` | 文案与 trace 均清空 |

## 变更记录

| 日期 | 相对旧版 | 原因 |
|------|----------|------|
| 2026-08-19 | 初版 | 与意图同步 |
