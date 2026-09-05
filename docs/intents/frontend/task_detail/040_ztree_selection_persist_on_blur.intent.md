# 意图：ztree 选中节点失焦后不自动隐藏关联面板

## 背景与目标

任务详情「任务关联」ztree（`div.layer-graph-ztree`）选中层/任务节点后，会展开对应的指令面板、文件树、层变更与执行日志。当前实现在 document 捕获阶段监听点击：点到任务关联面板外即 `emit('layer-graph-node-select', null)`，把选中清空，关联 UI 随之消失。用户在输入框失焦、点评论区其它位置、或切走焦点时，会误以为节点被取消。

目标：ztree 节点选中后，**失焦 / 点击面板外不得自动隐藏**对应元素；选中仅随用户点另一节点（或显式点虚拟/环节点清空）而切换。

## 范围与边界

- 范围内：`TaskDetailTaskLayerAssociationPanel` 的 document click 不再因面板外点击清空 `selectedLayerGraphNode`。
- 范围外：不改 layer-graph API、不改自动选中/refresh 策略（`layerGraphSelectionDismissedByUser` 仍用于显式 `select(null)`）；模型下拉 `<details>` 点外关闭保持不变。

## 约束与风险

- 模型选择下拉仍须点外关闭，避免挡住指令输入。
- 点击面板内（含指令 textarea）不得发出清空选中。
- 无选中时点击页面不得发出 `layer-graph-node-select`。

## 验收标准

1. 已选节点时，点击任务关联面板外 **不** 发出 `layer-graph-node-select`（尤其不得为 `null`）；指令面板 `comment-layer-ztree-command-panel` 仍因 props 保持可见。
2. 点击面板内不发出清空选中。
3. 无选中节点时点击页面不发出 select 事件。
4. 模型 `<details>` 打开时点击面板外：下拉关闭，且仍不取消选中。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| ztree 选中失焦后保持关联面板 | — | — | — | — | 纯前端交互，无服务端事实变更 |

## 实施计划

1. 将 click-outside 测例改为「不取消选中」；补「模型下拉仍关闭」。
2. 去掉 `onDocumentClick` 中对 `layer-graph-node-select` 的 `null` 派发；保留模型 details 点外关闭。
3. 跑通 `TaskDetailTaskLayerAssociationPanel.click-outside.test.js`。

## 变更记录

- 2026-08-17：任务详情页元素调整——ztree 失焦后不再自动隐藏对应指令/文件树/日志面板。
