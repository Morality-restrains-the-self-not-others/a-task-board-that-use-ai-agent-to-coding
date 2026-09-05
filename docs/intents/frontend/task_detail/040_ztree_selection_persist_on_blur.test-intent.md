# 测试意图：ztree 选中节点失焦后不自动隐藏关联面板

## 对应意图
`040_ztree_selection_persist_on_blur.intent.md`

## 测试目标

验证选中 ztree 节点后，点击任务关联面板外（失焦）不取消选中、不隐藏指令面板；模型下拉仍可点外关闭。

## 测试分层

- 单元：`taskFE/app/src/components/task-detail/TaskDetailTaskLayerAssociationPanel.click-outside.test.js`

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 已选层节点，指令面板可见 | 点击 `document.body`（面板外） | 不发出 `layer-graph-node-select`；`comment-layer-ztree-command-panel` 仍存在 |
| T2 | 同上 | 点击指令输入框 | 不发出 `layer-graph-node-select` |
| T3 | `selectedLayerGraphNode=null` | 点击页面 | 不发出 select 事件 |
| T4 | 已选节点且模型 `<details>` 已打开 | 点击面板外 | `details.open===false`，且不发出 `layer-graph-node-select` |

## 数据与环境

- Vitest + jsdom；`@vue/test-utils`；`attachTo: document.body`；无需网络。

## 通过标准

```
cd taskFE/app && npx vitest run src/components/task-detail/TaskDetailTaskLayerAssociationPanel.click-outside.test.js
```

全绿。
