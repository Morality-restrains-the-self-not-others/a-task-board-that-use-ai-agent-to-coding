# Completed OPT Archive — 2026-08-16

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 34 条。
> 归档执行时间：2026-08-19T16:47:13+08:00

## [OPT-20260815-017] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: CommentsPanel/Section/TaskDetail 死转发已清理，源码扫描单测守住
- **Created**: 2026-08-15
- **Context**: 去掉评论区全局镜像运行 header 后，`TaskDetailCommentsPanel` 仍 `defineEmits` `start-request-accepted` / `stop-server`，`TaskDetailCommentsSection` 仍向 Panel 监听并上抛。Panel 模板没有任何子组件发出这两类事件（停机按钮在评论「执行细节」运行态 Tab）。
- **Action**: (1) 从 `TaskDetailCommentsPanel.vue` 的 `defineEmits` 删除这两项；(2) 从 `TaskDetailCommentsSection.vue` 删除对应 `@start-request-accepted` / `@stop-server`；(3) 用源码扫描单测守住不再经 Panel 转发。
- **Why**: 死监听会让后续改动误以为运行态操作仍从评论列表顶端发出。
- **How to apply**: `taskFE/app/src/components/task-detail/TaskDetailCommentsPanel.vue`、`TaskDetailCommentsSection.vue`；停机入口在 `ServerConfigRuntimeStatusSection` 的 `comment-runtime-stop-server-btn`。

## [OPT-20260815-019] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 评论级进度生效收起任务级横幅，纯函数+接线单测守住
- **Created**: 2026-08-15
- **Context**: 评论执行细节「启动 TraceId」下方已展示项目克隆进度条；评论列表顶端在无关联仓库时仍可能显示「容器项目克隆进度」横幅，同一进度出现两处。
- **Action**: (1) 当 `cloneProgressByCommentId` 已能驱动至少一条评论进度条时，将 `showContainerCloneProgressBanner` 设为 false；(2) 补 vitest 覆盖「有评论级条则无顶端横幅」；(3) 保留关联项目区仓级条不改。
- **Why**: 双处进度会让用户以为有两轮克隆，且顶端横幅与评论卡位置距离远、对照困难。
- **How to apply**: `taskFE/app/src/views/taskDetailSectionBindings.js` 的 `commentsSectionProps.showContainerCloneProgressBanner`；对照 `TaskDetailCommentsPanel.vue` 的 `container-clone-progress-banner`。

## [OPT-20260815-020] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: collectRepoCloneJobs onSkippedNested 回调 + BOOTSTRAP_PHASE 用户可见启动日志，单测覆盖
- **Created**: 2026-08-15
- **Context**: 镜像已在 `auto_clone_nested_repos=false` 时 `appendOutboundReqLog('bootstrap-clone skip nested repos count=...')`，但任务详情「启动日志」折叠区仍主要展示克隆进度。用户关掉开关后若只看克隆行，容易误以为参数未生效。
- **Action**: (1) 在 `collectRepoCloneJobs` 跳过 nested 时同步写一条用户可见的启动日志（与排队/拉起实例同通道）；(2) 前端折叠区保留该行；(3) 单测断言 skip count>0 时日志文案出现。
- **Why**: 开关关闭是负向证据（没有子仓克隆），比「日志里多一行明确跳过」更难验收。
- **How to apply**: `trae-agent/onlineServiceJS/src/bootstrapRepoCredentials.mjs`；启动日志通道与 `正在启动容器实例` 同源。

## [OPT-20260815-022] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 辅助信息折叠块抽到 TaskDetailTaskAuxInfoPanel（500行），父组件121行，单测17项绿
- **Created**: 2026-08-15
- **Context**: 落地 OPT-20260815-021 时给身份面板加了 `commentId`，该文件已 563 行（行数门禁 500）。本次未整文件拆分以免扩大 comment_id 改动面。
- **Action**: (1) 把派生自/交付物类别/辅助信息等折叠块抽到已有 `useTaskIdentityPanelDisplay.js` 或独立子组件；(2) `wc -l` ≤ 500；(3) 现有 fork-from / aux-info / auto-run / deliverable-category 单测保持绿。
- **Why**: 继续往超标文件堆 prop 会反复触发行数门禁，且身份区已有多块独立 UI。
- **How to apply**: `taskFE/app/src/components/task-detail/TaskDetailTaskIdentityPanel.vue`；对照同目录 `*.aux-info.test.js` 等。

## [OPT-20260815-023] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: postJson 抽到 saasPostJson.mjs，saasTaskCloud 714→498 行，19 项单测全绿
- **Created**: 2026-08-15
- **Context**: 补齐容器出站 `comment_id` 时在 `postJson` 内合并 `withSaasInboundScope`，该文件已 714 行（行数门禁 500）。本次只加合并行，未整文件拆分以免把换票/心跳/层图推送搅进同一 diff。
- **Action**: (1) 把 `postJson` / 瞬时重试抽到独立模块（已依赖 `saasInboundScope.mjs`）；(2) 心跳与 layer-graph-push 可再分子文件；(3) `wc -l` 每个目标 ≤ 500；(4) 复跑 `saasTaskCloud.*.test.mjs` 全绿。
- **Why**: 继续往超标文件堆出站逻辑会反复触发行数门禁，且 `postJson` 已是所有 SaaS POST 的汇合点。
- **How to apply**: `trae-agent/onlineServiceJS/src/saasTaskCloud.mjs`；验收 `wc -l trae-agent/onlineServiceJS/src/saasTaskCloud.mjs` 以及 `node --test trae-agent/onlineServiceJS/src/saasTaskCloud.*.test.mjs` exit 0。

## [OPT-20260815-030] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 删除未引用 components/nestedRepoCloneStatusUtils 副本，统一 utils/ 单源
- **Created**: 2026-08-15
- **Context**: 修「bootstrapCloneDone 假完成」时发现 `taskFE/app/src/components/nestedRepoCloneStatusUtils.js` 与 `app/src/utils/nestedRepoCloneStatusUtils.js` 两份实现。生产 Vue 走 `utils/`（含日志段「已移入」回落）；`components/` 副本更短且缺 log-section 解析，两处需同步改逻辑。
- **Action**: (1) 确认无 Vue/测试仍 import `components/nestedRepoCloneStatusUtils.js`；(2) 删除该副本及其测试或改为 re-export `utils/`；(3) 只保留一份单测。
- **Why**: 双份状态推导会再次出现「任务级假完成 / 评论级漏仓」只修了一边。
- **How to apply**: `taskFE/app/src/components/nestedRepoCloneStatusUtils.js`、`taskFE/app/src/components/nestedRepoCloneStatusUtils.test.js`、`taskFE/app/src/utils/nestedRepoCloneStatusUtils.js`。

## [OPT-20260815-024] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 评论气泡作者行统一内联布局；删除未接线 CommentBubble 桩；CSS :where 特异性修复
- **Created**: 2026-08-15
- **Context**: 把头像移入昵称行后，`TaskDetailConversationFeed.vue` 父评论与子评论仍各写一遍作者行；同目录 `TaskDetailCommentBubble.vue` 是 2026-07 抽出的未接线占位（字母头像、`v-html` 原文）。
- **Action**: (1) 把作者行抽成单一子组件或真正接上 `TaskDetailCommentBubble`；(2) 删除未接线桩或改为唯一渲染入口；(3) `author-avatar-row` / `execution-details-embed` 单测保持绿。
- **Why**: 两套气泡模板会让下次改头像布局再次漏改子评论，未接线桩也会误导后续改动。
- **How to apply**: `taskFE/app/src/components/task-detail/TaskDetailConversationFeed.vue`、`TaskDetailCommentBubble.vue`。

## [OPT-20260815-027] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 拆分 ZTreeExecLogState 至 derived/layer-changes/live-output/watchers 模块，主文件 475 行
- **Created**: 2026-08-15
- **Context**: 修执行日志裸「404」时把 kv-last 换成 funcName-first helper，该文件仍约 969 行（行数门禁 500）。本次只改 URL/错误文案，未整文件拆分以免把层变动预取/轮询搅进同一 diff。
- **Action**: (1) 把层变动刷新/预取与 live output 派生抽到独立模块；(2) `wc -l` 每个目标 ≤ 500；(3) `taskDetailZTreeExecLogState.test.js` 保持绿。
- **Why**: 继续往超标文件堆执行日志逻辑会反复触发行数门禁，且该文件已同时承担状态、拉取与展示派生。
- **How to apply**: `taskFE/app/src/composables/taskDetail/taskDetailZTreeExecLogState.js`；对照同目录 `taskDetailZTreeExecLogState.test.js`。

## [OPT-20260815-026] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 私有父仓 nested enrich 无 identity 回退 owner_id；SQLite/HTTP 读 owner_id；容器快照同步
- **Created**: 2026-08-15
- **Context**: 修复「identities=0 则跳过子仓 enrich」后，公开 GitHub 父仓可走匿名 Contents API。私有仓匿名失败时仍会得到空 nested 列表，评论级继续只克隆元仓。
- **Action**: (1) `TaskSnapshot` 增加 `OwnerUserID`，SQLite `FetchTaskSnapshot` 读 `task_tasks.owner_id`；(2) `MergeNestedReposIntoSnapshots` 优先 identity userID，否则 owner_id；(3) HTTP `container-snapshot` 同步带 owner_id；(4) 补单测：identities 空 + owner 有值时仍能发现私有子仓。
- **Why**: 私有元仓（或私有 submodule）无法匿名列 `.gitmodules`，不回退 owner 则自动克隆开关对私有仓仍然无效。
- **How to apply**: `taskCredentialService/domain/entities.go`、`infrastructure/sqlite_business.go`、`application/services.go`、`taskTaskService/src/container_snapshot.go`。
- **Superseded note**: 2026-08-16 产品否定 owner_id 回退；改为评论 `created_by_id`（OPT-20260816-001）。OwnerUserID 字段已删除。

## [OPT-20260815-028] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 确认写入点 persistCommentBindingStartTraceID 已打 trace_id 结构化日志；Loki 查询可命中
- **Created**: 2026-08-15
- **Context**: 任务详情「启动 TraceId：d8b9efda4347cba2de274d64」在 Loki `{job=~".+"} |= "<id>"`（7 天）与 Tempo 均为 0 条；本机 `/var/log/runall` 也无命中。排查执行日志裸 404 时无法用该 ID 重建时间线。
- **Action**: (1) 确认页面启动 TraceId 的写入点（容器 bootstrap / 网关 start-job-stream）；(2) 在该路径打结构化日志，字段含 `trace_id` 与页面展示的同一 ID；(3) 用 Loki `{job=~".+"} | json | trace_id="<id>"` 能查到至少一条。
- **Why**: 页面已展示启动 TraceId，但日志未入库则 Agent/排障只能猜，无法走 TraceId 优先闭环。
- **How to apply**: Loki `http://10.2.150.68:3100`；对照任务详情 `[data-testid=comment-execution-start-trace-id]`。

## [OPT-20260816-001] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: nested 发现/克隆凭证改用 task_comments.created_by_id；删除 OwnerUserID 回退；Go 单测全绿
- **Created**: 2026-08-16
- **Context**: OPT-20260815-026 用 `task_tasks.owner_id` 回退私有仓 nested 发现。产品要求不允许该回退，容器执行必须用创建该评论的用户 git 身份。
- **Action**: (1) 从容器 token 的 comment_id 读 `task_comments.created_by_id`；(2) nested fetch 与克隆凭证只用该 user_id 的 `task_git_identities`；(3) 删除 OwnerUserID 回退；(4) 单测覆盖「任务 identity 属于他人时仍用评论作者」。
- **Why**: 评论作者与任务 owner 可能不是同一人；用 owner token 会越权克隆或对作者私有仓失败。
- **How to apply**: `taskCredentialService/application/nested_repos_enrich.go`、`application/services.go`、`infrastructure/sqlite_business.go`。

## [OPT-20260816-004] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: layer-oauth 按 token comment_id 解析评论作者 git 身份换票；单测覆盖他人身份时用评论作者 token；commit 9439f4d 已推送
- **Created**: 2026-08-16
- **Context**: nested 发现/克隆凭证已改用 `task_comments.created_by_id`。`LayerOauthService.ResolveLayerOauthTokens` 仍只读 `task_repo_identities`，PR 推送可能继续用任务绑定的他人身份。
- **Action**: (1) 从容器 token 取 comment_id；(2) 用评论作者 git identities 换票；(3) 补单测：任务 identity 属于他人时 layer-oauth userID 为评论作者。
- **Why**: 同一容器内克隆用作者身份、推送用 owner 身份会造成权限分裂和审计错位。
- **How to apply**: `taskCredentialService/application/layer_oauth.go`；对照 `layer_oauth_test.go`。

## [OPT-20260816-003] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 子仓库克隆状态重新克隆失败文案补 data-traceId（recloneErrorTraceIdByUrl 透传 + 失败 span 挂 data-traceId + 组件测 2 例）；ViewMode 已由 WIP 重构为只读无克隆状态，无需改动；commit 8bc1eea 已推送
- **Created**: 2026-08-16
- **Context**: 评论级克隆进度「手动重试」已写入 `recloneErrorTraceIdByUrl` 并挂 `data-traceId`。关联项目 / 子仓库克隆状态里同一套 `recloneStatusByUrl` 失败文案仍无 traceId。
- **Action**: (1) 把 `recloneErrorTraceIdByUrl` 传入 `TaskDetailNestedReposCloneStatus` 与 `TaskDetailLinkedProjectsViewMode`；(2) 失败 span 设置 `data-traceId`；(3) 补组件测。
- **Why**: 同一请求失败路径两处展示，只一处可跳 Loki。
- **How to apply**: `taskDetailFetchFns.js` `onRepoReclone` 已写 trace map；补 `TaskDetailNestedReposCloneStatus.vue` / `TaskDetailLinkedProjectsViewMode.vue` 的错误节点。

## [OPT-20260816-002] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 评论容器绑定提升至 useTaskDetail 单一实例，bindingStatusFor/CscIdFor 并入 commentForwardScope；CommentsSection 删除本地重复实例改收同一套函数 props；补测 running+CSC 优先于最后一条 AI 评论；commit ebd7f82 已推送
- **Created**: 2026-08-16
- **Context**: 执行日志/层变动已强制带 `comment_id`（无 id 不发请求）。但页面级 `containerForwardCommentId` 只根据 `displayComments` + `activeContainerAgentId` 解析，未注入 `bindingStatusFor` / `bindingCscIdFor`；CommentsSection 用绑定选「当前执行评论」。两评论绑不同 CSC 时，执行日志可能打到最后一条 AI 评论而非持有 CSC 的评论。
- **Action**: (1) 在 `useTaskDetail.js` 调用 `useCommentContainerBindings`，把 `bindingStatusFor` / `bindingCscIdFor` 并入 `commentForwardScope`；(2) 删除 `TaskDetailCommentsSection.vue` 内重复实例，改为接收同一套绑定函数；(3) 补测：有 CSC 的 running 评论优先于无绑定的最后一条 AI 评论；(4) 可选：`getContainerCompute` / `postContainerCompute` 在缺少 `comment_id` 时直接拒绝发请求。
- **Why**: 评论级 CSC 的隔离依赖正确的 `comment_id`；只保证「有 id」不够，选错评论仍会串台。
- **How to apply**: `useTaskDetail.js`、`TaskDetailCommentsSection.vue`、`resolveContainerUiContextCommentId.js`。验收：`cd taskFE/app && npx vitest run src/composables/taskDetail/taskDetailContainerFns.commentId.test.js src/composables/taskDetail/taskDetailExecLog.test.js src/composables/taskDetail/useCommentContainerBindings.test.js` 须 exit 0，且新增用例断言「running+CSC 评论优先于最后一条 AI 评论」。

## [OPT-20260816-005] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: taskDetailFetchFns.js 965→482 行：onRepoReclone 等抽至 taskDetailRepoReclone.js，选项类 fetch 抽至 taskDetailFetchOptions.js，主文件 re-export；taskDetail 全量单测 389 全绿；commit 82ea3b6 已推送
- **Created**: 2026-08-16
- **Context**: `taskFE/app/src/composables/taskDetail/taskDetailFetchFns.js` 已 965 行。本次只给 `onRepoReclone` 补了 traceId 写入，未做整文件拆分。
- **Action**: (1) 把 `onRepoReclone` 抽到 `taskDetailRepoReclone.js`；(2) 按 fetch/comment/edit 继续切模块直到各文件 ≤500；(3) 更新 `useTaskDetail.js` 引用并跑原测。
- **Why**: 行数门禁已触发；继续往该文件堆逻辑会让审查和回归定位更差。
- **How to apply**: `taskDetailFetchFns.js`、`useTaskDetail.js`；验收 `wc -l` ≤500 且 `npx vitest run src/composables/taskDetail/` 全绿。

## [OPT-20260816-006] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: ensure-client-ingress / 遗留启动轮询 URL 补 /comment_id/{cid}/ 路径；comment-container-bindings 与 previous-server-config 盘点为 task 级不加 comment_id
- **Created**: 2026-08-16
- **Context**: ADR-0010 已把执行日志、层图、Workbench、runtime-status、stop-vm、启动轮询的 `comment_id` 从 query 改到 path。仍有若干评论/任务云 URL 只带 `task_id` query：`comment-container-bindings`、`ensure-client-ingress`、`js/modal-task-detail-startup.js` 遗留轮询、`previous-server-config`。
- **Action**: (1) 盘点这些路径是否按评论 CSC 解析；(2) 若是，改用 `appendCommentIdPath` 并补测「无 `[?&]comment_id=`」；(3) 工作区级 API（machine-policy / runtime-indicators）保持不加 comment_id。
- **Why**: 评论级 CSC 漏带 id 会回落到任务级或 400「缺少评论ID」。
- **How to apply**: `commentExecutionApi.js`、`openContainerPage.js`、`modal-task-detail-startup.js`、`useServerConfigHardwarePanel.js`；对照 `commentIDFromComputeRequest`。

## [OPT-20260816-007] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: composer 预填最近一条评论 repo_identities，回退任务级；写入 commentRepoIdentityDraft；补组件测
- **Created**: 2026-08-16
- **Context**: 评论级身份已在「提交并运行」时强制校验并落库。每次 @镜像 仍要从空下拉重选，计划 I2 预填（最近评论 JSON → 否则 task_repo_identities）尚未做。
- **Action**: (1) 从当前任务评论列表取最近一条非空 `repo_identities`；(2) 否则读任务级 `task_repo_identities` 仅作预填；(3) 写入 `commentRepoIdentityDraft` 并选中下拉；(4) 补组件测。
- **Why**: 并行评论不再互相覆盖任务级表后，用户会觉得每次运行都要重选身份。
- **How to apply**: `CommentComposerRepoIdentity.vue`、`commentRepoIdentityDraft.js`；不要 POST 回写 `task_repo_identities`。

## [OPT-20260816-008] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: HTTPBusinessRepository.loadSnapshot 带 comment_id query，FetchTaskSnapshot/FetchTaskRepos 接口同步；parseContainerAPIPath 解析评论级路径；补 HTTP 单测
- **Created**: 2026-08-16
- **Context**: taskTaskService 已支持 `GET container-snapshot?comment_id=` 优先评论 JSON。taskCredentialService `HTTPBusinessRepository.loadSnapshot` 仍只按 task_id 拉，克隆可能读到任务级身份。
- **Action**: (1) `FetchRepoIdentities` / `loadSnapshot` 增加 comment_id 参数或 query；(2) 令牌签发与 start-vm 路径传入评论 ID；(3) 无 comment_id 时保持任务级回退；(4) 补 HTTP 单测。
- **Why**: 并行评论身份已按评论落库，凭据服务若不带 comment_id 会继续用错身份克隆。
- **How to apply**: `taskCredentialService/infrastructure/http_business.go`、`ports/repositories.go`；对照 `loadSnapshotRepoIdentities`。

## [OPT-20260816-009] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 看板评论卡接入 taskProjectsWithDetails + 拉取回退；showRunConfig 由 mention 驱动；补看板测
- **Created**: 2026-08-16
- **Context**: 任务详情 composer 已按关联仓库展示评论级身份。看板 `TaskCardCommentsSection` 未传 `taskProjectsWithDetails`，从看板 @镜像 运行会缺身份 UI，提交会被 400。
- **Action**: (1) 从看板任务卡片把关联项目详情传入 CommentsSection/Composer；(2) 无详情时拉任务项目；(3) 补看板测：有仓库且 mention 后出现 `comment-composer-repo-identity`。
- **Why**: 详情页与看板发评入口不一致，用户会以为看板不能选身份。
- **How to apply**: `TaskCardCommentsSection`、看板任务卡绑定；对照 `taskDetailSectionBindings.js`。

## [OPT-20260816-010] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: RegisteredTask.CommentID；cloudTokenAPIPrefix/statusPushURL 拼 /comment/{cid}/；补 token/push 测
- **Created**: 2026-08-16
- **Context**: UserData / `expandEnvForRuntime` 已把容器 `TaskApiEndPoint` 写成 `…/task/{task}/comment/{cid}/cloud`。`go_relayToTrae` 自己的 `cloudTokenAPIPrefix` 与 `statusPushURL` 仍只用 origin+tenant/workspace/task，走旧 `…/task/{task}/cloud`（网关仍兼容）。
- **Action**: (1) 给 `RegisteredTask` 增加 `CommentID`（从 env `COMMENT_ID` 或注册 body）；(2) `cloudTokenAPIPrefix` / `statusPushURL` 有 cid 时拼 `/comment/{cid}/`；(3) 补 `token_test.go` / `push_test.go`。
- **Why**: 评论级 CSC 下 relay 换票/status-push 若只走任务级 path，两评论同任务时可能串到错误 inbound 作用域。
- **How to apply**: `go_relayToTrae/src/token.go` `cloudTokenAPIPrefix`、`push.go` `statusPushURL`、`state.go` `RegisteredTask`。

## [OPT-20260816-011] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 加宽交付物过滤栏内容下拉 w-[7.5rem]→min-w-[7.5rem] w-56；补组件测关闭态可见 #序号；vitest 3 绿；taskFE c2945d8 提交推送
- **Created**: 2026-08-16
- **Context**: 过滤栏 `.deliverable-trail-level` 固定 `w-[7.5rem]`。option 已改为 `#N 标题`，关闭态原生 select 仍会截断标题，只保证编号在开头可见。
- **Action**: (1) 按最长常见 `#N` + 标题估算加宽或改为 `min-w` + `max-w`；(2) 核对换行后过滤栏不把看板顶出视口；(3) 补组件测：关闭态可见 `#` 序号。
- **Why**: 同名任务靠编号区分后，用户仍想扫一眼标题确认选了哪条。
- **How to apply**: `DeliverableBreadcrumb.vue` `.deliverable-trail-level`；`DeliverableBreadcrumb.test.js`。

## [OPT-20260816-012] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 对齐看板卡片编号 Playwright：mock 补 workspace_seq、断言 #序号、补 forward-auth 302 文档拦截 + /api 兜底、DOM 顺序改 compareDocumentPosition；headless 3 绿；taskFE da3983a 提交推送
- **Created**: 2026-08-16
- **Context**: `WorkPanel.task-card-id-badge.playwright.test.js` 仍断言 snowflake 后 6 位，且 mock todo 无 `workspace_seq`。`TaskCardIdBadge` 已改为 `formatTaskDisplayNo(workspace_seq)`，无序号时不渲染徽章。
- **Action**: (1) mock 补 `workspace_seq`；(2) 断言 `#N` 而非后六位；(3) 跑该 Playwright 确认全绿。
- **Why**: 现网卡片编号契约已切换，旧 E2E 会假红或因徽章不出现而超时。
- **How to apply**: `taskFE/tests/WorkPanel.task-card-id-badge.playwright.test.js`；对照 `TaskCardIdBadge.vue`、`taskIdDisplay.js`。

## [OPT-20260816-013] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 新增 mock 用例：@镜像 后临时硬件不齐仍可发评（noAutoRegion 空地域）→ hint 可见、comments POST 发出无 server_run_template；顺带修 zones/regions mock URL 前缀 + users/me mock；headless 2 绿；taskFE c2945d8 提交推送
- **Created**: 2026-08-16
- **Context**: 单元测已保证不完整临时规格不拦截 `submitComment`，且 composer 显示「无法启动运行」。现有 Playwright `TaskDetail.start-vm-auto-mock-ui` 只覆盖选齐实例后 body 含完整 `server_run_template`。
- **Action**: (1) 在同一 mock 套件加用例：@镜像后点临时配置但不选实例；(2) 断言可见 `comment-composer-hardware-run-hint`；(3) 点提交后 comments POST 发出、无 `server_run_template`、请求未被前端吃掉。
- **Why**: 真实面板的「临时配置」切换与提交按钮可点性只有 E2E 能一起看见。
- **How to apply**: `taskFE/tests/TaskDetail.start-vm-auto-mock-ui.playwright.test.js`；对照 `CommentComposerHardwareCard.vue`、`taskDetailFetchFns.js` `submitComment`。

## [OPT-20260816-014] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: orders.go 按职责拆分为 orders.go(471)/order_payment.go(139)/task_post_quota.go(282)/task_post_renewal.go(167)，19 符号全保留，go test ./src 全绿。顺带闭环同项 WIP：任务帖默认定价 33→1 元 + 续存同价（taskBill 4e5b0e4 + dataMigrate e8c098b 迁移039，均推送）。task-bill 已登记精准编译重启。
- **Created**: 2026-08-16
- **Context**: 将任务帖默认定价改为 1 元时改了 `consumeTaskPostRenewal` 的续存单价回退。`taskBill/src/orders.go` 当时已 1034 行，超过源文件 500 行门禁；本次只改 4 行，未做拆分以免把计价变更做成订单模块大重构。
- **Action**: (1) 按职责把 `orders.go` 拆成下单/支付状态、配额消耗、续存等文件，每个 ≤500 行；(2) 保持 `package main` 与现有 `*_test.go` 通过；(3) `wc -l taskBill/src/orders*.go` 验收。
- **Why**: 超标文件继续堆逻辑会让计费路径更难审、更容易在续存/配额上引入回归。
- **How to apply**: 入口 `consumeTaskPostQuota` / `consumeTaskPostRenewal` 在 `taskBill/src/orders.go`；拆分后跑 `go test ./src -count=1`。

## [OPT-20260816-015] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 009 建表补齐 COLLATE=utf8mb4_unicode_ci + 新增 022 幂等迁移统一存量库 collation；回归测试 test_recommended_llm_providers_collation.py；已应用 task_cloud（1 applied 21 skipped）并推送 dataMigrate 936acc3
- **Created**: 2026-08-16
- **Context**: 调整推荐供应商种子时核对 `009_recommended_llm_providers.sql`，建表只有 `DEFAULT CHARSET=utf8mb4`，未声明 `COLLATE=utf8mb4_unicode_ci`，与字符集元规则不一致。
- **Action**: (1) 新增 `dataMigrate/taskCloudService/022_*.sql`：`ALTER TABLE cloud_recommended_llm_providers CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`；(2) 回头给 009 的 CREATE TABLE 补上 COLLATE，避免新库与存量库排序规则漂移。
- **Why**: 不补齐时，该表可能继承服务器默认 collation；若与其它 utf8mb4_unicode_ci 表 JOIN/比较会触发 collation 冲突。
- **How to apply**: 文件 `dataMigrate/taskCloudService/009_recommended_llm_providers.sql`；新迁移接在 021 之后编号 022。

## [OPT-20260816-016] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 公司/工作空间/个人四条写路径 forceSubTokenDisabled 强制关闭 use_sub_token/budget_enabled；回归单测覆盖公司写入与个人创建；taskCloudService 3f75f7a
- **Created**: 2026-08-16
- **Context**: 租户环境变量页已隐藏「启用派生子Key」，保存路径会把 `use_sub_token` 写成 false。直接 POST `/api/cloud/feature-params/...` 仍可打开派生子 Key。
- **Action**: (1) 在 `taskCloudService` feature-params 写入路径把 `use_sub_token`/`budget_enabled` 规范为 false（或 400 拒绝）；(2) 补回归单测：请求带 true 时落库为 false 或被拒。
- **Why**: 仅前端隐藏挡不住 API 调用，已启用的租户配置也不会被服务端关掉。
- **How to apply**: `taskCloudService/src/feature_params_public_handlers.go` 的 provider 规范化；租户页 `WorkspaceSettingsFeatureParams.vue`。

## [OPT-20260816-017] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: MemberList.vue 预算引导文案去掉「启用派生子 Key」勾选提示；taskFE 2b0f67c
- **Created**: 2026-08-16
- **Context**: 派生子 Key 开关已从环境变量页拿掉，`MemberList.vue` 仍提示去该页勾选「启用派生子 Key」与「启用 LLM 预算」。
- **Action**: 改 `taskFE/app/src/components/MemberList.vue` 未启用预算时的说明，不再要求勾选已隐藏的开关。
- **Why**: 引导与当前页面能力不一致，管理员会找不到勾选入口。
- **How to apply**: `MemberList.vue` 中 `llmBudgetEnabled` 为 false 的分支文案。

## [OPT-20260816-018] cancelled

- **Status**: cancelled
- **Completed**: 2026-08-16
- **Summary**: 扩轮询否决；改为容器推送 + 刷新按钮对账（comment-runtime-push-sync / v84）
- **Created**: 2026-08-16
- **Context**: 执行面板已按评论快照隔离展示，但 `useServerConfigRuntime` 的 runtime poll / starting fallback 仍用 `scopedComment.forPoll()` 只刷新最近一次操作的 comment_id；Workbench/VS Code 链接仍可能来自任务级单槽。并行评论展开后，非最近评论的运行态会停在首次挂载拉取。
- **Action**: (1) 收集 `starting`/`running` 的 comment_id 列表 (2) poll 循环对每个 id 调用 `fetchServerRuntimeStatus` 写入对应 snapshot (3) 补 vitest：两评论并发 poll 后各自 status 独立更新
- **Why**: 否则长时间开着的第二条评论运行态会过期，看起来又像单例。
- **How to apply**: `taskFE/app/src/composables/taskDetail/useServerConfigRuntime.js` 的 `runtimeStatusPoll` 与 `startingFallbackPollId`；快照写入已有 `createCommentRuntimeSnapshotStore`

## [OPT-20260816-023] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: CLOUD_SERVER_STOPPED SSE 带 comment_id + runtime_status=Stopped；遗留看板 startup poll 改为 no-op
- **Created**: 2026-08-16
- **Context**: v84 停止命令路径已用 `publishTaskSSE` 注入 `comment_id`，进度文案带 `runtime_status=Stopping`。真正停机完成走 Kafka `CLOUD_SERVER_STOPPED`（`taskEvents/cloudserverstopped`），该路径尚未把 `Stopped` 推回 taskSSE，前端面板会停在 Stopping，只能点刷新对账。
- **Action**: (1) 在 `cloudserverstopped` 成功释放后 `publishTaskSSE` 带 `comment_id` + `runtime_status=Stopped`（及可被 `isCloudServerStopSuccessMessage` 识别的文案）(2) 补 Go 测：handler 成功后 SSE payload 含这两字段 (3) 前端测：该 SSE 写入对应评论 snapshot 且不跟 GET Describe
- **Why**: 直播通道要求状态变化当下 push；否则停机完成仍依赖按钮，和「只接收下发」不一致。
- **How to apply**: `taskEvents/internal/handlers/cloudserverstopped/handler.go`；`taskCloudService/src/compute_stop_vm.go` 已注入 comment_id 可复用；前端 `applyPushedRuntimeSnapshot`

## [OPT-20260816-037] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: createCommentLayerPanelStore 按 comment_id 分片层图/endpoint/执行日志/命令框；文件树 layer_id 只取该槽；A 的 404/空层不改 B；019 命令框一并验收。
- **Created**: 2026-08-16
- **Context**: 每条已挂 CSC 的评论都渲染执行细节，但 `layerGraphSnapshot`、`containerEndpointRegistered`、`layerExecLog*` 仍是页面级单例；文件树却按该评论 `comment_id` 转发。本轮已把 `layer not found` / 缺 `server_url` 改成等待态，避免红错，但跨评论仍可能共用选中节点与日志。
- **Action**: (1) 将层图快照、endpoint 旗标、执行日志 state 改为 `Record<commentId, …>` 或下放到 `TaskDetailCommentLayerAssociationBody` (2) 文件树 `layer_id` 只取该评论自己的层图 (3) 补测：评论 A 的 404 不影响评论 B 的树/日志 (4) 与 OPT-20260816-019 的命令框分片一并验收
- **Why**: 等待态只掩盖暂态；共享单例在多评论并行启动时仍会把别人的层 ID 打到本评论容器。
- **How to apply**: `useTaskDetail` / `taskDetailExecLog.js` / `taskDetailContainerFns.js` / `TaskDetailCommentsSection.vue` 的 `buildLayerBodyBind`；测例放 `taskFE/app/src/composables/taskDetail/`

## [OPT-20260816-019] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 命令框改为 patch-layer-panel + 每评论槽 commandText/Kind，两 body 输入互不影响。
- **Created**: 2026-08-16
- **Context**: 去单例后面，每个已挂 CSC 的评论都会挂 `TaskDetailCommentLayerAssociationBody`，但仍共用父级 `layerGraphCommandKind/Text` 等 v-model。在评论 A 输入命令会同步出现在评论 B。
- **Action**: (1) 把 layer 命令相关 state 改成 `Record<commentId, …>` 或下放到子组件 (2) 父级 expose 的 focus 仍指向当前操作评论 (3) 补测：两个 body 输入互不影响
- **Why**: 连接/运行态已按评论隔离，layer 命令框仍是任务级单槽，会让用户以为面板还是单例。
- **How to apply**: `TaskDetailCommentsSection.vue` 的 `TaskDetailCommentLayerAssociationBody`；state 现来自 `TaskDetail.vue` / `useTaskDetail` 的 layerGraph* models

## [OPT-20260816-039] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: container_layer_changes 按 comment_id/层归属写入槽；live output / agent step / 层变更面板从该槽派生，A 的 chunk 不进入 B。
- **Created**: 2026-08-16
- **Context**: 037 已把层图快照、endpoint、命令框、clone/job 错误槽按 comment_id 分片。`container_layer_changes` SSE 仍写入页面级 `layerChangesByLayerId`；`layerLiveOutputDisplay` / `layerAgentStepCards` / copyable 仍由页面级 zlog 派生，多评论同时有 job 流时展示可能串台。
- **Action**: (1) `container_layer_changes` 按 comment_id 或 layer 所属槽写入 (2) live output / agent step 卡片从该评论槽的 `liveOutputMap` + `jobExecutionPayload` 派生 (3) 补测：A 的 job-stream chunk 不出现在 B 的执行日志区
- **Why**: 037 覆盖了树和命令框；日志正文与层变更列表仍是共享派生，并行评论时用户会看成同一份输出。
- **How to apply**: `updateServerStatus.js` 的 `container_layer_changes`；`taskDetailZTreeExecLogLiveOutput.js` / `layerPanelViewFromSlot`

## [OPT-20260816-059] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: SaaS inbound 与网关只接受 /task/{id}/comment/{cid}/cloud；无 comment 段 404。
- **Created**: 2026-08-16
- **Context**: 已禁止生成/解析旧 `TaskApiEndPoint` `…/task/{taskId}/cloud`。网关、凭证面、Cloud inbound 仍匹配无 comment 段的 HTTP 路径（大量 handlers 测试）。env 契约已收紧，路由层仍接受旧 inbound。
- **Action**: (1) 盘点 gateway / taskCredentialService / taskCloudService / taskAgentSupport 对 `/task/{id}/cloud/` 的路由 (2) 改为必须 `/comment/{cid}/` 或 404 (3) 更新对应 HTTP 单测，不再把无 cid 路径当合法容器 inbound
- **Why**: 容器已无法发出旧前缀，残留路由会让手搓旧 URL 绕过 CSC。
- **How to apply**: `taskCredentialService/interfaces/handlers.go` `handleContainerAPI`；Cloud `handleContainerInboundToken`；gateway 前缀路由；本会话已收紧的 env 生成点不要回退

## [OPT-20260816-063] completed

- **Status**: completed
- **Completed**: 2026-08-16
- **Summary**: 已拆除跨任务闲置复用（ADR-0013）：删除 tryReuseIdleMachine/bindSharedMachineToTask；start-vm 仅白名单+同评论 inflight 附着；prefer_idle_reuse 恒 false；策略 UI 去掉勾选；闲置回收改为按本行 last_runtime_status。
- **Created**: 2026-08-16
- **Context**: `tryReuseIdleMachine` 用 `loadCloudServerConfig`（`comment_id=''`）把 instance 写到任务模板，并用 `isMachineNodeStarted` 读任务级 `last_runtime_status`。ADR-0007 后运行态只在评论 CSC，模板行 instance/status 已清空，现网评论机几乎不会被判为可复用；即便误复用也不会落到目标评论卡。同评论 inflight 附着仍有效。
- **Action**: (1) 候选改为评论 CSC：`idle_since` + 空 `server_url` + 非 Starting + 同 workspace + 同 `image_invoker_user_id` (2) `bindSharedMachineToTask` 写入目标 `comment_id` CSC，解绑源评论 CSC（禁止写模板行）(3) 复用后 `ModifyInstanceAttribute` 把 InstanceName 改成目标评论名，否则 heal/orphan 按名找不到 (4) 补单测：评论级闲置可复用、任务级模板不可复用、跨评论不抢 inflight
- **Why**: 继续走旧路径会让 prefer_idle_reuse 形同虚设（全部冷启动），或把机器写回任务级导致串台。
- **How to apply**: `taskCloudService/src/workspace_machine_idle_reuse.go`；`workspace_machine_recycle.go` 的 `isMachineNodeStarted` 同源；意图 `idle_reuse_same_workspace_user` / `idle_reuse_boot_guard_orphan_cross_check`；ADR-0007

