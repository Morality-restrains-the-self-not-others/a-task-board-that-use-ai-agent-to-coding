# 意图：任务详情层图 / 端点 / 执行日志 / 命令框按 comment_id 分片

## 背景与目标

每条已挂 CSC 的评论都渲染执行细节，但层图快照、容器 endpoint 旗标、执行日志与命令框曾是页面级单例。文件树却按该评论 `comment_id` 转发，等待态只能掩盖暂态；多评论并行启动时仍可能把别人的层 ID 打到本评论容器，或把评论 A 的命令同步到 B。

## 范围与边界

- 范围内：`createCommentLayerPanelStore`、`buildLayerBodyBindForComment`、SSE `container_layer_graph` / ui-context、HTTP `fetchContainerTaskUiContext` / `refreshLayerGraphFromServer`、命令框 `patch-layer-panel`、文件树 `layer_id` 取自该评论槽。
- 范围外：不为每条评论实例化整套 layer-graph watcher；不改 CSC 绑定模型；不新增进程内 `setInterval`。

## 约束与风险

- 无 `comment_id` 的 SSE 仍可回退页面级 / 当前执行评论，避免单评论任务回归。
- 切任务必须 `layerPanelStore.clear()`。
- 等待态（036）保留；本意图解决共享单例，而非再掩盖红错。

## 验收标准

1. 评论 A 的层图 404 / 空层 / 日志错误不改写评论 B 的 `snapshot`、文件树 `layer_id`、执行日志。
2. 评论 A 命令框输入不出现在评论 B。
3. 文件树 `selectedLayerGraphFileTreeLayerId` 只来自该评论槽，不取页面级共享层 ID。
4. 与 OPT-20260816-019 命令框分片一并验收。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 按评论隔离层图面板 | — | — | — | — | 纯前端 state 分片；SSE 已有 `comment_id` 时写入对应槽 |

## 实施计划

1. 槽 store + bind 纯函数与单测。
2. SSE/HTTP 按 `comment_id` 写入槽；UI 每条评论 `v-bind` 该槽。
3. 命令框改为 `patch-layer-panel`，不再共用 defineModel。
