# 意图：评论执行细节把 ztree（任务关联）提升为独立 Tab

## 背景与目标

任务详情评论「执行细节」展开后，Tab 栏目前是「执行细节 | 服务器运行状态」。ztree（`data-testid="comment-layer-ztree-panel"`，标题「任务关联（可写层串行 · 容器推送）」）嵌在「执行细节」面板内部，与依赖模式、容器连接、启动状态挤在同一屏。

目标：把 ztree **单独做成 Tab**，放在 Tab 栏**最左侧（上面）**，默认选中，展开执行细节即可直接操作可写层树。

## 范围与边界

- 范围内：`TaskDetailCommentExecutionDetails` 新增 `layerZtreeTab` + 具名插槽 `layer-ztree`；`TaskDetailCommentsSection` 将 `TaskDetailCommentLayerAssociationBody` 从默认插槽挪到该 Tab；容器名 / CSC / 启动 TraceId 仍在所有 Tab 外共享。
- 范围外：不改 layer-graph API、不改 Teleport、不新增接口或 Kafka 事件；无 ztree 可挂载时（未挂接 CSC）不出现该 Tab。

## 约束与风险

- Tab 顺序固定：`任务关联` → `执行细节` → `服务器运行状态`（后者仅 `serverRuntimeStatusTab` 时出现）。
- 仅 `layerZtreeTab` 时也显示 Tab 栏（任务关联 | 执行细节）。
- 两者皆关时保持旧行为：无 Tab 栏，默认插槽（及未启用时的 `layer-ztree` 插槽）直接展开可见。
- 默认激活：有任务关联 Tab 时为 `ztree`，否则为 `details`（既有 Playwright / 单测仍看得到 ztree）。
- 禁止 Teleport；就地插槽。

## 验收标准

1. 启用 `layerZtreeTab` 时，`comment-execution-tablist` 第一项为「任务关联」（`comment-execution-tab-ztree`），默认面板为 `comment-execution-panel-ztree`，内含 `layer-ztree` 插槽（即 ztree）。
2. 点击「执行细节」后 ztree 面板隐藏，默认插槽内容可见；再点「任务关联」ztree 回来。
3. 同时启用服务器运行状态 Tab 时，切到该 Tab 仍可见 `comment-execution-container-meta`；ztree 不在该面板内。
4. `layerZtreeTab=false` 且 `serverRuntimeStatusTab=false`：无 Tab 栏，行为与改前一致。
5. 评论区 fallback 与每条已挂接 CSC 的评论：`TaskDetailCommentLayerAssociationBody` 只出现在 `#layer-ztree`，不再混在默认插槽。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 将 ztree 放到独立 Tab | — | — | — | — | 纯前端展示重组，无服务端事实变更 |

## 实施计划

1. ExecutionDetails：prop + 具名插槽 + 第一 Tab；默认 `activeTab=ztree`。
2. CommentsSection：fallback 与 per-comment 的 association body 迁入 `#layer-ztree`，并打开 `layer-ztree-tab`。
3. 单测覆盖 Tab 顺序、默认面板、切换与无 Tab 回退；源码契约断言 `#layer-ztree` 出现两次。

## 变更记录

- 2026-08-16：任务详情页元素调整——将 ztree 从「执行细节」面板内提升为独立 Tab（最左、默认）。
