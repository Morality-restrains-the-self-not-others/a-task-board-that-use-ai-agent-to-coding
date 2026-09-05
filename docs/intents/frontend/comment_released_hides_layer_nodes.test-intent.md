# 测试意图：服务器已释放后隐藏任务关联节点面板

对应功能意图：`comment_released_hides_layer_nodes.intent.md`

## 测试点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 非服务 + 心跳暂停 + `layerGraphNodeCount>0` | `shouldShowCommentLayerZtreeReleased===true`，`shouldShowCommentLayerZtreeLoading===false` |
| T2 | binding `running` + runtime `Released` + 槽内仍有 zNodes | `shouldShowCommentLayerZtreeReleased===false`（走只读层图，不出释放空态/loading） |
| T3 | 同上，Body 还带已选节点与 file-tree layerId | DOM：有只读 `comment-layer-ztree-panel` 与 `comment-layer-ztree-exec-log-panel`；无 `layer-files-tablist` / `comment-layer-ztree-command-panel`；仍拉 SaaS job 日志，不拉 clone-log |
| T4 | binding `running` + runtime `Running` + zNodes | 不出释放空态（运行中不误伤） |
| T5 | 从未启动 idle、空树 | 不出释放空态 |

## 自动化落点

- `taskFE/app/src/utils/commentLayerZtreeUiState.test.js`
- `taskFE/app/src/composables/taskDetail/taskDetailCommentsSectionHelpers.test.js`
- `taskFE/app/src/composables/taskDetail/useCommentExecutionContext.test.js`
- `taskFE/app/src/components/task-detail/TaskDetailCommentLayerAssociationBody.released-hides-nodes.test.js`

## 不测

- 公网 Playwright 真机释放（依赖云实例生命周期；由单测锁定门控）
- 释放后层图快照是否从 store 删除（本意图只藏 UI）
