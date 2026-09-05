# 测试意图：层变更 SSE / 执行日志正文按 comment_id 分片

## 对应意图
`039_comment_layer_live_output_shard.intent.md`

## 测点

| ID | 测点 | 类型 | 位置 |
|---|---|---|---|
| T1 | live output / step 卡纯函数按 jobId+payload 派生 | 单元 | `taskDetailZTreeExecLogLiveOutput.test.js` |
| T2 | bind/view：A 的 chunk 与 step 卡不进入 B；页面级泄漏被 overlay | 单元 | `commentLayerPanelBind.test.js` |
| T3 | `commentIdOwningLayer` 按 snapshot.layers 归属 | 单元 | `commentLayerPanelStore.test.js` |
| T4 | `container_layer_changes` 写入 A 槽不改 B | 单元 | `updateServerStatus.test.js` |
| T5 | job-stream chunk 写入 A 的 liveOutputMap，B 的 view 不含该正文 | 单元 | `updateServerStatus.test.js` |

## 成功标准
T1–T5 vitest 全绿。
