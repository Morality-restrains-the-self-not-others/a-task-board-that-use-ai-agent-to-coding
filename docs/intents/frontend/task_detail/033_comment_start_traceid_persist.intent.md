# 意图：评论执行细节冷打开展示独立启动 TraceId

## 背景与目标

list API 增加 `start_trace_id` 后，前端冷打开须按评论回填，且不得用任务级 `statusTraceId`（常等于 `task_id`）冒充。

## 范围与边界

- 范围内：`useCommentContainerBindings` 冷打开回填；评论卡 `start-trace-id`；HTTP 受理体优先 `trace_id`。
- 范围外：不改 Tab 布局；空串仍不渲染 TraceId 行。

## 约束与风险

- 列值优先于日志后缀；等于当前 `taskId` 的值丢弃。
- 同任务两评论必须能展示两个不同 TraceId。
- 禁止 Teleport。

## 验收标准

1. list 返回 `start_trace_id` 时 `bindingStartTraceIdFor` 为该值，即使日志里有另一个 `trace_id=`。
2. 仅有任务级 `statusTraceId=task_…` 且等于/冒充任务 ID 时，评论 `startTraceId` 为空。
3. `buildStartVmAcceptedStatusUpdate` 优先响应体 `trace_id` 而非入站 `_traceId`。
4. SSE 带 `comment_id`+`trace_id` 时即使 message 为空，仍写入该评论 per-binding 启动 TraceId。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 展示本评论启动 TraceId | — | — | — | 纯前端展示 |

## 变更记录

- 2026-08-16：SSE `comment_id`+`trace_id` 在 message 为空时仍写入 per-binding 缓存（第二条评论启动中缺 TraceId）。
- 2026-08-13：列优先；禁用 task_id 回退。
