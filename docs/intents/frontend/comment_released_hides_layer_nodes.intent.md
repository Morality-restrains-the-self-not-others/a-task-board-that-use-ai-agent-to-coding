# 意图：服务器已释放后隐藏任务关联节点面板

- **日期**: 2026-08-22
- **状态**: 已实施
- **关联**: `comment_execution_details` 目标 10–11；测试意图同名 `.test-intent.md`

## 背景与目标

任务详情评论「执行细节」summary 在云实例已释放时已把徽章改为「服务器已释放」。但 `shouldShowCommentLayerZtreeReleased` 在 `layerGraphNodeCount > 0` 时直接返回 false，缓存的可写层 zTree、项目文件树 / 文件变动 Tab、已选节点指令面板仍留在「任务关联」里，用户无法对已释放机器操作这些节点。

目标：徽章或运行态进入已释放后，任务关联区若有层图快照则保留只读 zTree，并隐藏会打已释放容器的交互（指令/文件树/打开容器）。历史步骤走 SaaS `container-job-execution-log`（COS `step_full.json`），已选节点仍展示只读执行日志；仅跳过 clone-log 与层变动 prefetch。无快照时展示释放空态。

## 范围与边界

- 范围内：
  - `shouldShowCommentLayerZtreeReleased` / `shouldShowCommentLayerZtreeLoading` 感知 `executionReleased`
  - `resolvePerCommentLayerZtreeUi` 用与 summary 徽章同一套判定（binding + 该评论 runtime 快照）
  - `TaskDetailCommentLayerAssociationBody` 已有 `v-if` 释放空态优先于层图面板，靠正确 flag 生效
- 范围外：不新增 API / 领域事件；不在释放后清空后端层图快照；启停按钮仍在「服务器运行状态」Tab

## 约束与风险

- 与徽章对齐：`commentExecutionBindingBadgeText === '服务器已释放'` 时强制空态，即使 `isServerRunning` 仍为滞后 true、层图节点未清。
- Stopped 单独不改徽章文案（既有 T20）；若心跳暂停或 `serverRuntimeNotServing`，仍走非服务空态并隐藏节点。
- 从未启动且 heartbeat `idle`、无暂停/非服务信号 → 不出释放横幅。
- 容器仍 Running 且徽章为「容器 运行中」→ 缓存节点照常展示。

## 验收标准

1. runtime Released + 缓存节点 + 已选层 → 可见只读层图面板与执行日志/步骤卡片，不可见文件树 Tab / 指令面板 / 打开容器；仍拉 SaaS job 日志，不拉 clone-log。
2. binding `released` 且无层图快照 → 可见释放空态。
3. runtime Running + 缓存节点 → 仍展示层图与 Tab。
4. 从未启动 idle → 不误出释放空态。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 已释放时隐藏层图交互节点 | — | — | — | — | 纯前端展示门控；沿用既有 runtime SSE / binding，无新领域事件 |

## 变更记录

| 日期 | 相对旧版 | 原因 |
|------|----------|------|
| 2026-08-22 | 初版 | 释放后缓存节点仍可点选文件树与发指令 |
| 2026-08-23 | 有快照保留只读 zTree，隐藏交互并跳过日志拉取 | 快照仍可复查；打已释放容器会 409 |
| 2026-08-23 | 释放后仍拉 SaaS/COS 步骤并展示只读日志 | ADR-0039 后 job 日志不依赖活容器；先前「跳过拉取」导致步骤空白 |
| 2026-08-25 | 释放后不得折叠掉 exec-log 面板；watcher 仍 hydrate COS | 折叠横条导致 step_full 归档日志对用户不可见 |
