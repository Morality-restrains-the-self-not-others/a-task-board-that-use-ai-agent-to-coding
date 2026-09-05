# 意图：评论级执行细节与容器状态归属

## 背景

任务详情将容器连接状态与「任务关联（可写层）」从 Runtime 区 / 评论区全局顶部，迁入**对应评论**的可折叠「执行细节」，并支持依赖模式持久化与评论级容器绑定调度（OPT-038/039）。

## 目标

1. 从 `TaskDetailServerStartStatusPanel` 抽出 `TaskDetailContainerConnectionStatus`；Runtime 区保留服务器启动与 SSE。
2. 评论 Feed 每条顶层评论（user/ai）下增加可折叠「执行细节」。**零评论时不挂任务级空 comment-id fallback**（避免「串行/当前执行」误导）；启服跳过原因用 `TaskDetailAutoRunSkipBanner`。
3. 每条评论各自一份执行面板（Tab / 容器元信息 / 运行态 / 连接状态），展示该评论自己的 CSC、容器名、启动 TraceId 与 runtime-status；**禁止**把完整面板做成任务级单例、只挂在 `activeExecutionCommentId` 上。ztree（任务关联）为独立 Tab，见 `task_detail/035_comment_ztree_independent_tab`。
4. `isActive` / `activeExecutionCommentId` 仅用于「当前执行」徽章与默认展开，不作为面板内容门控。
5. 依赖模式：`wait_previous`（默认）/ `independent`；UI 可切换，PATCH 持久化到 TTS / taskAIComment。
6. 评论容器绑定：taskCloudService `comment_container_bindings`；前端 ensure + advance；每条评论展示自己的 binding 状态。
7. `waiting_previous`（「容器 等待前序」）须在 Tab 中提供「前序评论」：点开后再看摘要与执行状态（`listCommentPredecessors`，无新 API / 无轮询）。默认不占执行细节面板空间。
8. `waiting_previous` 的「服务器运行状态」不得展示 stray CSC Running：覆盖为「等待前序」，隐藏启停按钮；`bindingIsStarting` 不含 `waiting_previous`；per-comment 启动面板不得传入任务级 `runtimeStatus`。
9. 执行细节 summary 展示该评论发评时选定的 Git 身份（`repo_identities[].git_identity_id` 解析为身份标签，如「默认身份」）；无选身份的评论不展示该徽章。
10. 执行细节 summary 的容器状态徽章必须与该评论「服务器运行状态」一致：云实例 `Released` / 展示「已释放」或 binding 已为 `released` 时，徽章文案为 **「服务器已释放」**，不得再显示「容器 运行中」。binding 刷新滞后时以运行态快照为准。
11. 当徽章为「服务器已释放」（或机器已非服务：Released / terminated / 心跳暂停）时：若仍有层图快照则**保留只读 zTree**，并隐藏打开容器 / 刷新 / 文件树 Tab / 指令输入与发送等会打已释放容器的交互。**执行步骤走 SaaS**（`GET container-job-execution-log` 水合 COS `step_full.json` / 事件表，ADR-0039），已选节点时仍展示只读执行日志与步骤卡片；仅跳过打容器的 clone-log 与层变动 prefetch（避免 409）。无快照时展示释放空态。重新启动后由既有 SSE/刷新恢复写交互。

## 非目标

- 云上真正一评论一 ECS 多实例的后端 UNIQUE 迁表（前端已按 comment_id 分流展示）
- 用后台 Describe 轮询扫全部/最近 comment_id（已否决，见 `comment_runtime_no_background_poll`：仅前端触发或 SSE 推送）

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 依赖模式变更 | CommentExecutionModeChanged | CommentExecutionModeChanged | 前端 PATCH → taskTaskService / taskAIComment | Feed 回放；调度 ensure | 证据豁免：前端仅发起 PATCH，领域事件由 Go 服务发布 |
| 评论容器绑定推进 | CommentContainerBindingAdvanced | CommentContainerBindingAdvanced | 前端 ensure/advance → taskCloudService | 执行细节 binding 状态 | 证据豁免：前端仅调用 API，领域事件由 taskCloudService 发布 |
| 评论执行细节 UI 重组 | — | — | — | — | 纯前端展示 |

后端事件 SSOT：`docs/intents/backend/cloud/comment_container_bindings.intent.md`。

## 变更记录

- 2026-07-22：初版（goal-mode 锁定方案 A）
- 2026-07-22：OPT-038/039 — 持久化 + 绑定调度接线
- 2026-08-13：启动日志 UTC/本地时钟对齐并按规范化正文去重，见 `comment_startup_log_timezone_dedupe`
- 2026-08-15：去掉评论列表顶端全局「镜像运行状态」header（`image-runtime-entry`）；启动/SSE/运行态只留在本评论执行细节内
- 2026-08-16：执行面板去单例——每条评论独立 Tab/运行态/连接状态，`isActive` 不再门控完整面板
- 2026-08-16：ztree 从「执行细节」面板内提升为独立「任务关联」Tab（最左、默认），见 `035_comment_ztree_independent_tab`
- 2026-08-16：`waiting_previous` 前序评论改入独立 Tab（与「执行细节 / 服务器运行状态」并列），点开再查阅，避免 nested details 占空间
- 2026-08-16：串行 `waiting_previous` 覆盖 stray CSC 运行态，避免「等待前序」评论显示服务器已启动
- 2026-08-17：ztree 选中节点失焦/点面板外不再自动隐藏指令面板，见 `040_ztree_selection_persist_on_blur`
- 2026-08-17：`auto_run` 软跳过启服须在评论区上方展示原因 +「强制重新启动」（见 `98_autorun_skip_no_ui_reason`）；空评论不得再挂伪「执行细节」fallback
- 2026-08-17：去掉零评论 `execution-details-fallback`（空 comment-id +「当前执行」）；执行细节仅随真实评论出现
- 2026-08-21：执行细节 summary 展示评论级 Git 身份（`repo_identities`）
- 2026-08-22：云实例已释放时 summary 徽章改为「服务器已释放」，不以滞后的 binding=`running` 显示「容器 运行中」
- 2026-08-22：徽章/运行态已释放时隐藏缓存的层图节点、文件树 Tab 与已选节点面板（`shouldShowCommentLayerZtreeReleased` 不再因 `layerGraphNodeCount>0` 短路）
- 2026-08-23：有层图快照时保留只读 zTree；隐藏指令/日志/文件树交互；`refreshZTreeExecutionLog` 在 `containerReleased` 时跳过拉取
- 2026-08-23：纠正「释放后不拉日志」：历史步骤来自 SaaS/COS，释放后仍拉 job 日志并展示步骤卡片；仅跳过 clone-log / 层变动 prefetch / 写交互
- 2026-08-25：释放后不得把 exec-log 折叠成「步骤来自归档」单行横条；watcher 在已释放时仍 hydrate COS step_full
- 2026-08-27：任务详情发评区将队列与执行依赖同卡；入队锁定 wait_previous。见 `comment_queue_serial_execution`
