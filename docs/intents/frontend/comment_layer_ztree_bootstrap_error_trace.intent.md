# 意图：任务关联引导克隆失败须带 data-traceId

- **日期**: 2026-08-19
- **状态**: 已实施

## 背景与目标

容器 `BOOTSTRAP_FAILED` 经 SSE `container_bootstrap_failed` 推到任务详情「任务关联」加载区，错误文案（含「引导克隆失败：仓库 Git 授权未齐」）原先没有 `data-traceId`。`updateServerStatus` 对该 status **提前 return**，从未把 `statusData.trace_id` 写入可绑定的 ref；且不得复用任务级 `statusTraceId`（启动链路常把该字段写成 `task_id`）。

目标：错误 `<p data-testid="comment-layer-ztree-loading-error">` 在 SSE 带 `trace_id` 时挂载 `data-traceId`，无则省略，禁止 `"unknown"`。

## 范围与边界

- 范围内：taskCloudService runtime-event → SSE `trace_id`（body / fields / HTTP ctx）；taskFE 独立 `containerBootstrapFailureTraceId` → ztree 错误节点。
- 范围外：不把 task 级 `statusTraceId` 回退到评论执行细节；自动运行跳过横幅（冷打开无原始请求 trace，除非另落库）。

## 约束与风险

- 属性名固定 `data-traceId`；缺失时省略属性。
- 令牌无效等非引导失败错误不得挂引导 SSE 的 trace。

## 验收标准

1. SSE `container_bootstrap_failed` + `trace_id` → `containerBootstrapFailureTraceId` 等于该值，且不写 `statusTraceId`。
2. 错误节点 `[data-testid=comment-layer-ztree-loading-error]` 的 `data-traceId` 等于该值。
3. SSE 无 `trace_id` 时错误节点不带 `data-traceId`。
4. `BOOTSTRAP_COMPLETE` 或层图出现后清空文案与 trace。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 引导失败展示 trace | SSE_MESSAGE | Kafka SSE | taskCloudService `publishBootstrapRuntimeOutcomeSSE` | taskFE 任务详情 | 复用既有 SSE，无新领域事件 |

## 变更记录

| 日期 | 相对旧版 | 原因 |
|------|----------|------|
| 2026-08-19 | 初版 | 引导克隆失败文案无法按 trace 查 Loki |
