# 意图：任务详情层变更 SSE / 执行日志正文按 comment_id 分片

## 背景与目标

037 已把层图快照、endpoint、命令框、clone/job 错误槽按评论隔离。`container_layer_changes` SSE 仍写入页面级 `layerChangesByLayerId`；`layerLiveOutputDisplay` / `layerAgentStepCards` / `layerJobOutputDisplay` 仍由页面级 zlog 派生。多评论同时有 job 流时，评论 B 会显示评论 A 的日志正文或层变更列表。

## 范围与边界

- 范围内：`container_layer_changes` 写入该评论槽；`layerPanelViewFromSlot` 从该槽的 `liveOutputMap` + `jobExecutionPayload` + `layerChangesByLayerId` 派生展示；无 `comment_id` 时按层所属槽回退。
- 范围外：不为每条评论实例化 zlog watcher；不改后端 SSE 契约；不新增 `setInterval`。

## 约束与风险

- 无 `comment_id` 且无法从层/job 归属解析时，仍可写页面级，保证单评论任务不回归。
- 缺槽时 overlay 空派生（空字符串 / 空数组），禁止回落到页面级 live output。

## 验收标准

1. `container_layer_changes` + `comment_id=A`（或 layer 属于 A）不改写 B 的 `layerChangesByLayerId`。
2. A 的 job-stream chunk 不出现在 B 的 `layerLiveOutputDisplay`。
3. A 的 agent step 卡片不出现在 B 的 `layerAgentStepCards`。
4. 页面级 `layerLiveOutputDisplay` 泄漏值不得进入无槽/他槽 bind。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 按评论隔离层变更与执行日志正文 | — | — | — | — | 纯前端派生与 SSE 槽写入 |

## 实施计划

1. 抽出 live output / step 卡纯函数；`layerPanelViewFromSlot` overlay。
2. store `commentIdOwningLayer`；SSE `container_layer_changes` 写入槽。
3. 补测 A 不影响 B。
