# 测试意图：评论执行细节 ztree 独立 Tab

## 对应意图
`035_comment_ztree_independent_tab.intent.md`

## 测试目标

验证 ztree 作为独立「任务关联」Tab 出现在 Tab 栏最左且默认选中；切走后不出现在执行细节 / 服务器运行状态面板；无 Tab 模式不回归；评论区把 association body 挂到 `#layer-ztree`。

## 测试分层

- 单元：`taskFE/app/src/components/task-detail/TaskDetailCommentExecutionDetails.ztree-tab.test.js`
- 源码契约：`taskFE/app/src/composables/taskDetail/commentExecutionPanelPolicy.test.js`

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | `layerZtreeTab=true`，`serverRuntimeStatusTab=true`，两插槽均有内容 | 默认渲染 | Tab 顺序为 任务关联 → 执行细节 → 服务器运行状态；默认可见 `comment-execution-panel-ztree`；执行细节 / 运行状态面板不存在 |
| T2 | 同上 | 点「执行细节」 | ztree 面板隐藏；`comment-execution-panel-details` 存在；再点「任务关联」ztree 回来 |
| T3 | 同上，含容器元信息 | 点「服务器运行状态」 | 运行状态面板存在；`comment-execution-container-meta` 仍在；ztree 面板不存在 |
| T4 | 仅 `layerZtreeTab=true` | 默认渲染 | 有 Tab 栏；无「服务器运行状态」Tab；默认 ztree 面板 |
| T5 | 两 Tab prop 均为 false | 展开 | 无 tablist；默认插槽可见；若提供 `layer-ztree` 则出现在执行细节面板内 |
| T6 | CommentsSection 源码 | 静态读取 | `#layer-ztree` 出现 2 次（fallback + per-comment）；含 `layer-ztree-tab` |

## 数据与环境

- Vitest + jsdom；`@vue/test-utils`；无需网络。

## 通过标准

```
cd taskFE/app && npx vitest run src/components/task-detail/TaskDetailCommentExecutionDetails.ztree-tab.test.js src/components/task-detail/TaskDetailCommentExecutionDetails.test.js src/composables/taskDetail/commentExecutionPanelPolicy.test.js
```

全绿。
