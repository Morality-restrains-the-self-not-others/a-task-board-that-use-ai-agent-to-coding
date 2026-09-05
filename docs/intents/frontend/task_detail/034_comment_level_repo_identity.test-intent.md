# 测试意图：评论级仓库提交身份与授权身份

## 对应功能意图

`docs/intents/frontend/task_detail/034_comment_level_repo_identity.intent.md`

## 用例

| ID | 场景 | 前置 | 步骤 | 期望 |
|----|------|------|------|------|
| T1 | 关联项目只读 | 任务已关联 1 项目 1 仓库 | 打开任务详情非编辑 | 见项目名/URL/分支；无 OAuth 绑定、保存账号、Git 提交身份、克隆进度、重新克隆、拉取/同步身份 |
| T2 | 编辑关联仍可用 | 同上 | 进入编辑 | 可改关联项目与分支 |
| T3 | 纯评论无身份区 | at-mode 开或关 | 不 @镜像 | 无 `comment-composer-repo-identity`；提交不带 `repo_identities` |
| T4 | 运行评论展示身份 | 有关联仓库、at-mode | @镜像 | 出现每仓 Git 身份下拉；GitHub 仓有账号选择 |
| T5 | 缺身份不拦截、缺 OAuth 拦截 | T4 | 不选身份但 OAuth L1+L2 已齐：可 POST。未绑 L1、或 L1 已绑但无 session `grant_ticket` 点提交并运行 | L1+L2 已齐：仍发 POST（body 含 `grant_ticket`）。未齐：不 POST；`comment-composer-oauth-blocked-reason` 可见；提交按钮 disabled |
| T6 | POST 带身份 | T4 选齐 | 提交并运行 | body `repo_identities` 含 repo_url + git_identity_id（GitHub 另含 github_user_id）；有 session ticket 时含 `grant_ticket` |
| T7 | 持久化回读 | T6 成功 | GET 评论列表 | 该评论 `repo_identities` 与提交一致 |
| T8 | snapshot 优先评论 | 任务级表与评论 JSON 不同 | GET container-snapshot?comment_id= | 返回评论 JSON |
| T9 | snapshot 回退 | 旧评论无 JSON | GET snapshot 无 comment 或空 JSON | 返回 task_repo_identities |
| T10 | 事件增补 | T6 | 消费 TASK_COMMENT_IMAGE_MENTIONED | payload 含 repo_identities |
| T11 | @镜像 OAuth 提示并拦截 | 有关联 GitHub/GitLab 仓、用户未绑 OAuth 或未完成本条评论使用授权 | @镜像 | 出现 `comment-composer-git-oauth-hint` data-kind=unbound，文案含使用授权且**不含**「仍可发送评论」；含 `a[href*="github-start-from-gateway" 或 "gitlab-start-from-gateway"]` 且带 `repo_url` 与 `grant_kind=pending`；提交并运行按钮 disabled |
| T12 | 运行评论展示子仓开关 | 有关联仓库且 project 含 git_repos | @镜像 | 身份行出现 `task-nested-repos-auto-clone-toggle`；默认跟随 `project.auto_clone_nested_repos`（缺省 true） |
| T13 | 关闭子仓开关 | T12 开关为 ON | 关掉开关 | PUT `/api/projects/tenant_id/{tid}/{projectId}/` body `auto_clone_nested_repos=false`；出现 `task-nested-repos-auto-clone-off-hint`；失败须 `data-traceId` |
| T14 | 容器 task-detail 评论级身份 | 任务级表 gid-task，评论 JSON gid-comment 且 git_identities 有 name/email | 容器凭该评论 token 拉 task-detail | `repo_git_identities` 为 gid-comment 的 name/email，不是 gid-task 或作者其它 identity |
| T15 | snapshot 含作者名 | 同 T8 且 identity 行存在 | GET container-snapshot?comment_id= | `user_name`/`user_email` 来自 `task_git_identities` |
| T16 | 执行细节回显身份 | T7 评论已落库 | 看该评论执行细节 summary | `comment-execution-git-identity` 文案含所选身份标签 |
| T17 | 执行细节回显 Git OAuth | 关联 GitHub/GitLab 仓且任务级 OAuth 已检查 | 看该评论执行细节 summary | `comment-execution-git-oauth` 与 Git 身份同一行；已绑定文案含「已绑定」；未绑定含「未绑定」且 `comment-execution-git-oauth-bind` 为真实 `a[href*="start-from-gateway"]` |
| T17b | 已绑定但 push 无仓库写权限 | OAuth 芯片本会显示已绑定；层快照 `last_push_error` 含 `Permission to … denied` | 看该评论执行细节 summary 与 zTree 层节点 | `comment-execution-git-oauth` 为 `bound_no_write`、文案「Git OAuth · 无写权限」；`comment-execution-git-oauth-bind` 文案含「换账号授权」且为真实 `a[href*="start-from-gateway"]`；zTree `layer-ztree-push-error-label` 为「push 无权限」 |
| T18 | 评论区显示探测 AccessToken | 评论区可见且任务级检查为已绑定 | 打开任务详情 | 对已绑定仓只发 **一次** `probe_access_token=1`；无效则摘要改「未绑定」；无 `setInterval`；未绑定不换票 |
| T18c | 非内网 GitLab 探测网络不可达 | DB 已绑定；`probe_access_token=1` 返回 `network_status=unreachable`（即使 cache 仍有 token） | 看该评论执行细节 summary | `comment-execution-git-oauth` 为 `unreachable`、文案「Git OAuth · 网络不可达」；无 `comment-execution-git-oauth-bind`；不得显示已绑定 |
| T18d | 内网 GitLab 探测跳过不可达 | 租户已标记内网；平台探测 GitLab 超时 | 看该评论执行细节 summary | `network_status=skipped_intranet`；芯片保持已绑定，不显示网络不可达 |
| T19 | 去绑定回流后芯片已绑定 | 执行细节显示未绑定；用户完成 GitLab OAuth 并带回 `gitlab=ok` | 回到任务详情 | 任务级查询 `user-app-connection`（可有限次重试）；`comment-execution-git-oauth` 为已绑定；`gitlab` query 被清掉且保留 `accessCode` |
| T20 | accessCode 页绑定探测 401 | 任务详情带 `accessCode`、连接接口 401 会话失效 | 打开任务详情 | 仍停留 task-detail，不跳 `/auth/login/`；芯片为未绑定或检查中 |
| T21 | 共用 AccessToken 不并存已绑定/去绑定 | 执行细节芯片已绑定；PR 状态曾返回授权失效 | 看该评论执行细节与 PR 回复卡 | `comment-execution-git-oauth` 为已绑定；`comment-git-pr-oauth-bind` 不出现；不展示「授权已失效」内联文案 |
| T22 | 服务端拒绝无 L2 的 @镜像评论 | L1 connected、无 grant_ticket、关联 GitHub HTTPS 仓 | POST comments 带 @镜像 | 400 `code=git_oauth_comment_grant_missing`；带有效 `grant_ticket` 则 201 且 `repo_identities[].oauth_gitsite` 已打标 |
| T23 | 云端开发 commit 前校验评论 L2 | 运行评论无 `oauth_gitsite`、远端为 GitHub | 层图「提交并推送」 | 不调用 git commit；前端提示使用授权；`comment_oauth_grants` 缺失时不得用 comments 切片误判已授权 |

## 回归

- `TaskDetailLinkedProjectsPanel.test.js`：删除「应显示 OAuth 绑定入口文案」等与只读冲突的断言，改为只读断言。
- `useLinkedProjectsRepoOAuth.test.js`：面板挂载即查 `user-app-connection`；`gitlab=ok` 有限次重试后 allBound 并清 query、保留 accessCode。
- Playwright `TaskDetail.comment-git-oauth-return-bound`：拦截连接接口为已绑定后 `comment-execution-git-oauth` 为 bound；拦截 401 时 accessCode 页不跳登录。
- `CommentExecutionGitOauthBadge.test.js` / `commentExecutionGitOauth.test.js` / `layerZtreePushError.test.js`：T17b 已绑定 + Permission denied → `bound_no_write` + zTree「push 无权限」。
- Playwright `TaskDetail.*oauth-bind*` / `relay-oauth-start-blocked`：OAuth 主路径在创建/编辑任务；composer 仅为补救；relay 未绑定引导指向创建/编辑任务。
- 评论克隆进度单测保持在评论执行细节，不回到关联项目。
- `TaskDetailCommentComposer.oauthGate.test.js`、`taskDetailFetchFns.imageMention.test.js`、`taskDetailLayerActions.test.js`、`gitOauthPushPrecheck.test.js`、`commentOAuthGrantCheck.test.js`：未绑 L1 或不带 grant_ticket 不 POST 评论、不 commit/push。
