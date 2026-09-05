# 测试意图：创建/Fork 任务时完成仓库 Git OAuth 绑定（自动运行）

## 对应功能意图

`docs/intents/frontend/create_task_oauth_bind.intent.md`

## 用例

| ID | 场景 | 前置 | 步骤 | 期望 |
|----|------|------|------|------|
| T1 | 无 OAuth 仓 | 所选项目无 GitHub/GitLab URL | 打开创建弹窗 | 无 OAuth 提交拦截 |
| T2 | 未绑定拦截（auto_run） | 所选项目有 GitHub 仓，`auto_run=true`，user-app-connection `connected:false` | 打开创建弹窗 | 提交 disabled；blocked-reason 含 OAuth；出现 `create-task-repo-oauth-bind` |
| T2b | 未勾选 auto_run 不拦截 | 同仓未绑定但 `auto_run=false` | 打开创建弹窗 | 无 OAuth 提交拦截；无 `create-task-repo-oauth-bind` |
| T3 | 已绑定可提交 | 同仓项目 L2 已打且 validate-git-repos `token_available`，`auto_run=true`，其它门禁已满足 | 打开创建弹窗 | 无 OAuth 拦截；出现 `create-task-repo-oauth-bound` |
| T3c | 仅 L1 无项目 L2 仍拦截 | `user-app-connection connected:true`，无 session ticket，validate-git-repos `not_bound` | 打开创建弹窗 auto_run | 提交 disabled；出现 bind 链接 |
| T3d | 项目详情已授权可提交 | 同用户刚在项目页完成 `grant_kind=project`，工作面板对该项目 auto_run | 打开创建弹窗 | `create-task-repo-oauth-bound`；无 bind 链接 |
| T3e | 默认选中刚打开的项目 | 工作区列表第一项不是刚授权的项目；localStorage 记了详情页 project id | 打开创建弹窗 | `projectSelections[0].projectId` 为刚打开的项目，而非列表第一项 |
| T3b | Fork 未绑定拦截 auto-run | 任务关联 GitHub 仓，OAuth 未绑定 | 任务详情点 Fork，选择自动运行 | `fork-confirm-submit` disabled；`fork-auto-run-oauth-bind` 可见；切到仅派生后可确认 |
| T4 | 检查失败 | user-app-connection 非 2xx 且带 traceId | 打开创建弹窗 | 提交 disabled；原因含「无法检查」；错误节点 `data-traceId` |
| T5 | 克隆失败文案 | bootstrap 原文含 repo-clone-credentials | 渲染 ztree 提示 | 含「创建或编辑任务」；不含「添加评论」 |
| T6 | composer 补救 | 未绑 OAuth 的 GitHub/GitLab 仓 | 打开评论身份区并 @镜像 | `comment-composer-git-oauth-hint` 含「创建或编辑任务」，不含「仍可发送评论」；提交并运行 disabled；纯评论不拦截 |
| T7 | relay 未绑定引导 | startBlockedByUnboundOAuth | 打开直接启动面板 | `relay-to-trae-oauth-unbound-guide` 含「创建或编辑任务」，不含「添加评论区域」 |
| T8 | 区域 GitLab 克隆键 | provider YAML website 含 `${subdomains.gitlabTencentSh1}` | ResolveProvider(该 host) | 键为 `gitlab:tencent-sh-1`，不是 `gitlab:default` |
| T8b | Path A IP YAML miss 仍换票 | 仓 `http://115.29.110.74/example-user/somanyad.git`，YAML 无该 host，任务 identity.UserID>0 | BuildRepoCloneCredentials / layer-oauth | 不进 missing_repo_credentials；FetchAccessToken 键为 `gitsite:115.29.110.74`；provider=gitlab、git_http_username=oauth2 |
| T9 | 跨实例连接检查 | 只绑 `gitlab:tencent-sh-1` | GET user-app-connection 另一 GitLab `repo_url` | `connected:false` |
| T10 | 软跳过不建评论 | nested git / OAuth 探测失败 | 创建 auto_run 任务 | 无 ensureAutoRunAtComment；有 skip_reason 横幅字段 |
| T11 | 引导失败 data-traceId | SSE `container_bootstrap_failed` 带 `trace_id` | 渲染 ztree 错误节点 | `[data-testid=comment-layer-ztree-loading-error]` 有 `data-traceId`；无 trace 则省略 |
| T12 | 空层锚点仍展示凭证失败 | 层图含 empty/`bootstrap_pending` 且 clone-log 凭证不齐 | 任务关联 Tab | `comment-layer-ztree-panel` 可见；其内 `comment-layer-ztree-loading-error` 含「Git 授权未齐」；不可见「正在准备可写层」 |

## 可执行测试

- `taskFE/app/src/utils/createTaskOauthGate.test.js`
- `taskFE/app/src/utils/createTaskPreferredProject.test.js`
- `taskFE/app/src/composables/useCreateTaskRepoOAuth.test.js`
- `taskFE/app/src/components/CreateTaskModal.oauth-gate.test.js`
- `taskFE/app/src/components/task-detail/ForkAutoRunConfirmModal.test.js`
- `taskFE/app/src/components/task-detail/TaskDetailPageHeader.test.js`
- `taskFE/app/src/utils/commentLayerZtreeUiState.test.js`
- `taskFE/app/src/utils/commentRepoIdentity.test.js`
- `taskFE/app/src/utils/gitOauthPushPrecheck.test.js`
- `taskFE/app/src/components/task-detail/TaskDetailCommentComposer.oauthGate.test.js`
- `taskFE/app/src/composables/taskDetail/taskDetailFetchFns.imageMention.test.js`
- `taskFE/app/src/composables/taskDetail/taskDetailLayerActions.test.js`
- `taskFE/tests/TaskDetail.relay-oauth-start-blocked.playwright.test.js`
- `taskCredentialService/infrastructure/provider_configs_test.go`
- `taskCredentialService/application/clone_provider_test.go`
- `taskCredentialService/application/build_credentials_test.go`
- `taskCredentialService/application/layer_oauth_test.go`
- `taskCredentialService/infrastructure/gitoauth_client_test.go`
- `taskGitOauth/src/gitsite_access_for_user_test.go`
- `taskGitOauth/src/connection_handlers_test.go`
- `taskTaskService/src/auto_run_test.go`
- `taskFE/app/src/composables/taskDetail/containerBootstrapSse.test.js`
- `taskFE/app/src/composables/taskDetail/updateServerStatus.test.js`
- `taskFE/app/src/composables/taskDetail/taskDetailCommentsSectionHelpers.test.js`
- `taskFE/app/src/components/task-detail/TaskDetailCommentLayerZtreeStatus.test.js`
- `taskFE/tests/TaskDetail.writable-layer-credentials-stuck.playwright.test.js`
- `taskFE/app/src/utils/layerZtreeBootstrapAnchor.test.js`
- `taskFE/app/src/components/task-detail/TaskDetailTaskLayerAssociationPanel.bootstrap-error.test.js`
- `taskCloudService/src/container_runtime_event_test.go`
