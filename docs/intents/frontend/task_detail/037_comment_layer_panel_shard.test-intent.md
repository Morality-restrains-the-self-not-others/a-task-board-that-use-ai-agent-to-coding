# 测试意图：任务详情层图面板按 comment_id 分片

## 对应意图
`037_comment_layer_panel_shard.intent.md`

## 测点

| ID | 测点 | 类型 | 位置 |
|---|---|---|---|
| T1 | 两评论 snapshot / endpoint / exec log / 命令文本互不覆盖 | 单元 | `commentLayerPanelStore.test.js` |
| T2 | 文件树 layer_id 与 bind 不串台；页面级共享层 ID 不得泄漏 | 单元 | `commentLayerPanelBind.test.js` |
| T3 | `container_layer_graph` + `comment_id=A` 不改 B 的槽，也不覆盖非 active 页面级快照 | 单元 | `updateServerStatus.test.js` |
| T4 | `fetchContainerTaskUiContext(A)` 的 endpoint 不写入 B | 单元 | `taskDetailContainerFns.commentId.test.js` |
| T5 | 命令框 patch A 不改 B；listeners 带 comment id | 单元 | `taskDetailCommentsSectionHelpers.test.js` |

## 成功标准
T1–T5 vitest 全绿。
