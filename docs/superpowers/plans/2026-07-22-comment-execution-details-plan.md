# 评论级执行细节 — 实施计划

## 爆炸半径与测试缺口

- **触点**：`TaskDetail*.vue`、`useCommentExecutionContext`、Comments Feed 子树
- **不变**：SSE composable、layer graph API、container heartbeat 逻辑
- **缺口**：无 `useCommentExecutionContext` 单测；ServerStartStatusPanel 测试需拆分

## 任务

- [x] P1 新建 `TaskDetailContainerConnectionStatus.vue`（从 ServerStartStatusPanel 抽出）
- [x] P2 瘦身 `TaskDetailServerStartStatusPanel.vue`（移除容器连接块 + 更新测试）
- [x] P3 新建 `TaskDetailCommentExecutionDetails.vue`（折叠、依赖 badge、条件子组件）
- [x] P4 新建 `useCommentExecutionContext.js` + 单测（T1–T4、T8）
- [x] P5 改 `TaskDetailConversationFeed` / CommentItem 挂载 ExecutionDetails
- [x] P6 移除 `TaskDetailCommentsPanel` `#layer-association`；迁移 layer 至 active 评论
- [x] P7 无评论回退：CommentsPanel 顶部 `CommentExecutionDetails` 占位
- [x] P8 更新意图 / 测试意图；冒烟与组件单测（T5–T7）
- [x] P9 Review + Ship（架构 v51 已切换 current；见 review 文档）
