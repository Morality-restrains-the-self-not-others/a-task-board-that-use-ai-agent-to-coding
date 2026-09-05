# 测试意图：评论级执行细节

## 用例

### T1 — activeContainerAgentId 归属

- **给定** Feed 含 user 评论 C1 及其 container_agent A1，且 `activeContainerAgentId=A1`
- **当** 渲染评论 Feed
- **则** C1 的 ExecutionDetails 带「当前执行」徽章；C1 与其它顶层评论各自渲染独立执行面板

### T6 — 非 active 评论仍有完整本评论面板

- **给定** active 为 C1，用户展开 C2 的执行细节，且 C2 已挂接独立 CSC
- **当** 渲染 C2
- **则** C2 可见依赖 badge、本评论容器名/CSC、以及本评论的运行态 Tab；文案不得引导去「当前执行」评论看完整面板

### T14 — 两评论执行面板数据隔离（2026-08-16）

- **给定** Feed 有评论 C1、C2，各自 binding 为 running 且 CSC/容器名/启动 TraceId 不同
- **当** 同时展开两条「执行细节」
- **则** 两个 `comment-execution-details` 的 `data-comment-id` 不同；Tab 均存在；runtime 展示字段来自各自 snapshot，互不覆盖

### T2 — 无 agent 时最新 AI 评论

- **给定** 无 activeContainerAgentId，Feed 有 user C0 与 ai C2（C2 更新）
- **当** 计算 `activeExecutionCommentId`
- **则** 归属 C2

### T3 — 仅 user 评论

- **给定** 无 agent、无 AI 评论，仅有 user C3、C4（C4 更新）
- **当** 计算归属
- **则** 归属 C4

### T4 — 零评论不挂执行细节（2026-08-17）

- **给定** `displayComments` 为空
- **当** 渲染 CommentsPanel
- **则** **不**渲染 `comment-execution-details-fallback-wrap` / 空 `data-comment-id` 的「执行细节」；Feed 显示「暂无评论」
- **并且** 执行细节仅在有真实评论时挂在该评论下；`auto_run` 跳过原因由评论区上方 `TaskDetailAutoRunSkipBanner` 独立展示（见 T4b）

### T4b — auto_run 软跳过启服可见（2026-08-17）

- **给定** 任务 `auto_run=true` 且 `auto_run_start_skip_reason` 非空（软跳过）
- **当** 打开任务详情
- **则** 评论区上方可见 `auto-run-start-skipped-banner`，文案含跳过原因；可点 `auto-run-force-restart` 发 `PATCH force_auto_run=true`
- **并且** Fork 创建响应含 skip 字段时须弹「自动运行未启服」（与工作面板创建一致）
- **并且** 落库前存量：`auto_run=true` 且 `comments=[]` 时亦显示横幅（启发式）并提供强制重试

### T5 — Runtime 无容器连接块

- **给定** 容器 heartbeat 非 idle
- **当** 渲染 Runtime ServerStartStatusPanel
- **则** 无 `容器连接状态` 标题；SSE/启动状态仍可见

### T6b — 未挂接 CSC 的评论无 layer 命令

- **给定** active 为 C1，用户展开 C2 的执行细节，且 C2 无 `csc_id`
- **当** 渲染 C2
- **则** 可见依赖 badge；不可见 LayerGraph 命令面板

### T7 — 默认 wait_previous

- **给定** 新评论无后端 dependency 字段
- **当** 构建 CommentExecutionContext
- **则** `dependencyMode === 'wait_previous'`

### T8 — 依赖模式切换 PATCH

- **给定** 评论 C9 当前 `wait_previous`
- **当** 用户在执行细节点击「可并行」
- **则** 发出 `change-dependency-mode`；调用人类/AI PATCH；本地 `execution_mode=independent`

### T9 — 绑定状态展示与调度

- **给定** 任务评论列表非空
- **当** 加载 Feed
- **则** 对每条 ensure binding + advance；`waiting_previous`/`running` 等状态显示在执行细节 badge

### T16 — 等待前序可查看前序及执行状态（2026-08-16）

- **给定** 评论 C3 binding 为 `waiting_previous`，Feed 中 C1 运行中、C2 已完成（C3 为 wait_previous 且无显式 depends_on）
- **当** 渲染 C3 执行细节
- **则** Tab 栏出现 `comment-execution-tab-predecessors`（文案含「前序评论」与数量）；默认仍停留在「执行细节」Tab，**不**在 Tab 栏上方展开前序列表
- **并且** 单击「前序评论」Tab 或单击 summary 上「容器 等待前序」徽章后，可见 `comment-execution-predecessor-list`；两行分别展示摘要+状态；徽章含前序数量 `· N`
- **并且** 单击前序行发出 `focus-predecessor`（父级滚到该评论执行细节）；不发起新的后台轮询
- **并且** independent / 无前序且非 waiting_previous 时不渲染该 Tab / 列表

### T10 — 容器元信息展示启动 TraceId

- **给定** 评论执行细节已展开，且 `statusTraceId`/`startTraceId` 非空（start-vm 受理或 SSE `trace_id`）
- **当** 渲染 `comment-execution-container-meta`
- **则** 在容器名/CSC 下方可见「启动 TraceId」行；节点带 `data-testid=comment-execution-start-trace-id` 与 `data-traceId=<id>`
- **并且** `startTraceId` 为空时不渲染该行

### T11 — 服务器运行状态 Tab 也展示容器元信息

- **给定** `serverRuntimeStatusTab=true`，且容器名 / CSC / 启动 TraceId 均有值
- **当** 点击「服务器运行状态」Tab
- **则** `comment-execution-panel-server-runtime` 存在，且 `comment-execution-container-meta` 仍可见容器名、CSC、「启动 TraceId」（含 `data-traceId`）
- **并且** 切回「执行细节」后元信息仍在

### T12 — 启动日志同一步骤不因时区/trace_id 重复

见 `comment_startup_log_timezone_dedupe.test.intent.md`。

### T15 — ztree 独立「任务关联」Tab（2026-08-16）

见 `task_detail/035_comment_ztree_independent_tab.test-intent.md`。

### T18 — ztree 选中失焦后不隐藏关联面板（2026-08-17）

见 `task_detail/040_ztree_selection_persist_on_blur.test-intent.md`。

### T13 — 评论 Feed 顶端无全局镜像运行 header（2026-08-15）

- **给定** 评论区有评论且服务器运行中
- **当** 渲染 `TaskDetailCommentsPanel`
- **则** 无 `data-testid=image-runtime-entry` / `image-runtime-entries-section`
- **并且** 启动日志与 SSE 仍可在对应评论「执行细节」中查看

### T17 — waiting_previous 不得显示 stray CSC Running（2026-08-16）

- **给定** 评论 C_java binding=`waiting_previous`，但其 comment_id 下仍有 Running CSC（历史 @镜像旁路启机）
- **当** 绑定 `bindCommentRuntimePanel(panel, C_java, 'waiting_previous')`
- **则** 展示文案为「等待前序」，`showRuntimeActionButtons=false`
- **并且** `bindingIsStarting(C_java)===false`，生命周期为「等待前序」而非「启动中」

### T21 — 服务器已释放时隐藏交互并保留只读层图（2026-08-23）

- **给定** 评论执行细节徽章为「服务器已释放」（runtime `Released` 覆盖滞后 `running` binding，或 binding=`released`），且该评论槽仍缓存 `layerGraphZNodes`、已选节点与文件树 layerId
- **当** 渲染 `comment-layer-association-body` / `TaskDetailTaskLayerAssociationPanel`
- **则** 仍可见只读 `comment-layer-ztree-panel`；不可见 `layer-files-tablist`、`comment-layer-ztree-command-panel`、打开容器/刷新、指令输入与「发送给AI」
- **并且** 已选节点时仍可见 `comment-layer-ztree-exec-log-panel`（COS/SaaS 步骤卡片）
- **并且** `refreshZTreeExecutionLog` 在 `containerReleased` 时仍 `apiFetch` `container-job-execution-log`，且不得请求 `container-clone-log`；watcher 仍 `refreshZTreeExecutionLog`，但跳过层变动 prefetch
- **并且** 无层图快照时可见释放空态 `comment-layer-ztree-released`
- **并且** 机器仍在服务且徽章为「容器 运行中」时，缓存节点仍展示层图与交互（不误伤运行中评论）
- **并且** 从未启动且 heartbeat `idle`、无释放信号时，不因空节点误出释放空态（沿用既有 S2/never-started 契约）

### T20 — 服务器已释放时 summary 徽章不得显示「容器 运行中」（2026-08-22）

- **给定** 评论 C1 binding 仍为 `running`，该评论 runtime 快照 `status=Released`（或展示文案「已释放」）
- **当** 渲染 C1 的 `comment-execution-details-summary`
- **则** `comment-execution-binding-status` 文案为「服务器已释放」，不得含「容器 运行中」
- **并且** binding=`released`（即使尚无 runtime 快照）时同样显示「服务器已释放」
- **并且** runtime 为 Running 且 binding=`running` 时仍显示「容器 运行中」
- **并且** runtime 仅为 Stopped 时不把徽章改成「服务器已释放」

### T19 — 执行细节 summary 展示发评 Git 身份（2026-08-21）

- **给定** 评论 C1 的 `repo_identities` 含 `git_identity_id=gi_1`，身份目录中 gi_1 的 `label=默认身份`
- **当** 渲染 C1 的 `comment-execution-details-summary`
- **则** 可见 `comment-execution-git-identity`，文案含「Git 身份 · 默认身份」；hover title 为完整 name/email/label
- **并且** `repo_identities` 为空（如未选身份的自动运行评论）时不渲染该徽章
- **并且** 身份目录尚未加载时仍渲染「Git 身份」徽章，不伪造人名

## 自动化落点

- `task2app/front_project/app/src/composables/taskDetail/useCommentExecutionContext.test.js`
- `task2app/front_project/app/src/composables/taskDetail/useCommentContainerBindings.test.js`
- `task2app/front_project/app/src/utils/commentExecutionApi.test.js`
- `task2app/front_project/app/src/components/task-detail/TaskDetailContainerConnectionStatus.test.js`
- `task2app/front_project/app/src/components/task-detail/TaskDetailCommentExecutionDetails.test.js`
- `taskFE/app/src/composables/taskDetail/commentRuntimeSnapshotStore.test.js`
- `taskFE/app/src/composables/taskDetail/buildCommentRuntimePanelDisplay.test.js`
- `taskFE/app/src/composables/taskDetail/commentExecutionPanelPolicy.test.js`
- `taskFE/app/src/composables/taskDetail/bindCommentRuntimePanel.test.js`
- `taskFE/app/src/utils/startVmHttpResult.test.js`（启动受理写入 `trace_id`；响应体优先于入站 `_traceId`）
- `taskFE/app/src/composables/taskDetail/taskDetailCommentsSectionHelpers.test.js`
- `taskFE/app/src/components/task-detail/TaskDetailCommentsPanel.noGlobalImageRuntimeHeader.test.js`
- `task2app/front_project/app/src/components/task-detail/TaskDetailServerStartStatusPanel.test.js`（更新）
- `taskFE/app/src/composables/taskDetail/useCommentContainerBindings.test.js`（033：列优先、禁用 task_id 回退）
- `taskFE/app/src/utils/bindingServerStartupLogs.test.js`
- `taskFE/app/src/composables/taskDetail/createBindingStatusLogs.test.js`
- `taskFE/app/src/utils/commentCloneProgressFromLogs.test.js`（评论级克隆进度条，见 `comment_clone_progress_under_trace`）
- `taskFE/app/src/components/task-detail/TaskDetailCommentCloneProgress.test.js`
- `taskFE/app/src/composables/taskDetail/useCommentExecutionContext.test.js`（T16：listCommentPredecessors）
- `taskFE/app/src/components/task-detail/TaskDetailCommentPredecessorList.test.js`（T16）
- `taskFE/app/src/components/task-detail/TaskDetailCommentExecutionDetails.test.js`（T16：waiting_previous 列表）
- `taskFE/app/src/composables/taskDetail/bindCommentRuntimePanel.test.js`（T17：waiting_previous 覆盖 stray Running）
- `taskFE/app/src/composables/taskDetail/useCommentContainerBindings.test.js`（T17：bindingIsStarting 不含 waiting_previous）
- `taskFE/app/src/components/task-detail/TaskDetailTaskLayerAssociationPanel.click-outside.test.js`（T18：失焦不取消选中）
- `taskFE/app/src/utils/commentExecutionGitIdentity.test.js`
- `taskFE/app/src/components/task-detail/TaskDetailCommentExecutionDetails.test.js`（T19：summary Git 身份徽章）
- `taskFE/app/src/composables/taskDetail/useCommentExecutionContext.test.js`（T20：commentExecutionBindingBadgeText）
- `taskFE/app/src/components/task-detail/TaskDetailCommentExecutionDetails.test.js`（T20：Released 覆盖 running）
- `taskFE/app/src/composables/taskDetail/buildCommentRuntimePanelDisplay.test.js`（T20：快照带出 raw status）
- `taskFE/app/src/composables/taskDetail/bindCommentRuntimePanel.test.js`（T20：从 panel 取 runtime status）
- `taskFE/app/src/utils/commentLayerZtreeUiState.test.js`（T21 / VS-RSD-05：无快照才出释放空态）
- `taskFE/app/src/composables/taskDetail/taskDetailExecLog.test.js`（T21 / VS-RSD-06：released 仍拉 SaaS job 日志、跳过 clone-log）
- `taskFE/app/src/composables/taskDetail/taskDetailZTreeExecLogWatchers.test.js`（T21 / VS-RSD-06：released watcher 仍刷新 job 日志）
- `taskFE/app/src/composables/useTaskDetail.zlog-released.test.js`（T21：zlog 工厂传入 serverRuntimeNotServing）
- `taskFE/app/src/components/task-detail/TaskDetailTaskLayerAssociationPanel.released-hides-interactive.test.js`（T21：有快照隐藏写交互、保留 COS 步骤日志）
- `taskFE/app/src/composables/taskDetail/taskDetailCommentsSectionHelpers.test.js`（T21：Released 覆盖 stale running）
- `taskFE/app/src/components/task-detail/TaskDetailCommentLayerAssociationBody.released-hides-nodes.test.js`（T21：DOM 隐藏 Tab/已选面板）
- `taskFE/app/src/components/task-detail/TaskDetailCommentsSection.git-identity-summary.test.js`
- `taskFE/app/src/composables/taskDetail/buildDisplayComments.test.js`（保留 repo_identities）
