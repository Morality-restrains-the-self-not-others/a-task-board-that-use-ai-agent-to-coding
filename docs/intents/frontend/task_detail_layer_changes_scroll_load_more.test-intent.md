# 任务详情变动文件列表滚动加载更多 — 测试意图

| ID | 场景 | 期望 |
| --- | --- | --- |
| T1 | unit：`applyLayerParentDiffPagination` 分页 | `has_more`/`next_offset`/`change_count` 正确 |
| T2 | unit：`normalizeLayerChangesPayload` 保留 `change_count` | 分页首屏不把 total 压成 page length |
| T3 | unit：`mergeLayerChangesPage` | 按 path 去重追加 |
| T4 | unit：Hints `hasMore` | 文案含「向下滚动可加载更多」 |
| T5 | unit：Hints 仅 `truncated` | 文案含「目录扫描已达上限」 |
| T6 | unit：Hints `truncatedTraceId` | 节点 `data-traceId` 等于传入值；空则省略属性 |
| T7 | unit：`collectIndex` 跳过噪声目录 | `node_modules`/`.venv` 不进索引、不触顶 |
| T6 | unit：Preview `errorTraceId` | 错误节点带 `data-traceId`（既有） |
| T7 | Go：L0 `container-layer-diff-parent-files` | 转发 URL 含 offset/limit |
| T8 | Go：execution-log 拉 layer changes | 请求带 `offset=0&limit=100` |
| T9 | Django：`container-layer-diff-parent-files` | 在 `MIGRATED_OUTBOUND_ACTIONS`，stub 410 |

## 验证命令

```bash
cd trae-agent/onlineServiceJS && node --test src/layerParentDiff.pagination.test.mjs
cd taskContainerGateway && go test ./src -count=1 -run 'TestHandleContainerJobExecutionLog|TestL0RegistryLayerDiffParentFiles'
cd task2app/front_project/app && npx vitest run \
  src/composables/taskDetail/taskDetailZTreeExecLogState.test.js \
  src/components/task-detail/TaskDetailExecLayerChangesHints.test.js \
  src/components/task-detail/TaskDetailExecLayerChangePreview.test.js
```
