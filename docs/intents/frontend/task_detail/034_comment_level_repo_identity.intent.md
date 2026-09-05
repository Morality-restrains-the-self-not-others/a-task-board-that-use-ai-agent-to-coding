# 意图：评论级仓库提交身份与授权身份

## 背景与目标

容器启动已是评论级，但「关联项目」仍承担任务级 OAuth、Git 提交身份、克隆进度。并行评论会互相覆盖任务级身份表，进度也挂在错误生命周期上。

目标：关联项目只展示任务绑定的项目/仓库/基准分支；**本次运行**的 Git 提交身份与授权身份在添加「提交并运行」评论时确定，并随该评论持久化。

## 范围与边界

- 范围内：taskFE 关联项目 ViewMode/Toolbar 去掉身份与进度操作；composer 运行配置增加按仓库身份；`task_comments.repo_identities_json`；评论 POST/列表；container-snapshot `comment_id`；`TASK_COMMENT_IMAGE_MENTIONED` 增补。
- 范围内：容器 `task-detail` / clone credentials / layer OAuth 的提交者身份必须读该评论 `repo_identities_json`（经 credential `FetchRepoIdentities(task, comment)`），禁止用作者全部 `task_git_identities` 覆盖每仓选择。
- 范围内：评论 JSON 可含 `oauth_gitsite` / `oauth_remote_user_id` / `oauth_granted_at`（L2 使用标记）；无标记不得换票。创建自动运行走 grant_ticket，不继承项目 L2。
- 范围内：项目级「自动克隆子仓库」开关在 composer 身份行展示并 PUT `auto_clone_nested_repos`（仍是项目配置，不写入评论 JSON）。
- 范围外：不改 onlineServiceJS 克隆协议细节；不删除 `task_repo_identities` 表（回退+预填）；不把纯评论强制身份；不新增 Python API；不把子仓开关做成评论级字段。
- `@镜像` 时提示是否已绑定 Git OAuth（L1 user-app-connection）以及本条评论的使用授权（L2 / `grant_ticket`）。关联 GitHub/GitLab HTTPS 仓且未完成 **L2**（或 L1 未绑定 / 检查失败）时，**拦截「提交并运行」**（禁用按钮 + POST 前再校验 L1 与 session `grant_ticket`），服务端无 `oauth_gitsite` 拒绝创建评论；**纯评论**（无 `@镜像`）不拦截。私有仓缺凭证不得先创建评论再在推送阶段失败。

## 约束与风险

- 有 `@镜像` 的评论**建议**带齐任务全部 git_repos 的身份；缺 Git 提交身份不拦截 POST（前端）。缺 Git OAuth **L1 或评论 L2 / grant_ticket** 时拦截「提交并运行」。不得把账号中心已连接（L1）当成评论使用授权（L2）。
- 禁止运行评论写回任务级 identity/github binding（并行覆盖）。
- 旧评论无 JSON 时 snapshot 回退任务级表。
- 报错须带 `data-traceId`（请求失败路径；纯前端 OAuth 提示不得伪造 traceId）。

## 验收标准

1. `task-linked-projects-panel` 非编辑态无「OAuth 绑定」「保存账号」「Git 提交身份」「拉取当前层级」「同步到容器」、无克隆进度条、无重新克隆。
2. 非编辑态仍展示项目名、仓库 URL、基准分支。
3. composer 在 `showRunConfig` 时出现 `comment-composer-repo-identity`；无运行配置时不出现。
4. 身份行有 `project_id` 时出现 `task-nested-repos-auto-clone-toggle`；切换 PUT `/api/projects/tenant_id/{tid}/{projectId}/` 的 `auto_clone_nested_repos`；关闭时出现 `task-nested-repos-auto-clone-off-hint`。保存失败须带 `data-traceId`。
5. 「提交并运行」请求可含 `repo_identities`（有选则带）与 `grant_ticket`（本会话使用授权）。未选 Git 提交身份不拦截发送。关联 GitHub/GitLab 仓且未绑定 Git OAuth **或未完成本条评论 L2** 时：**拦截**发送（`comment-composer-oauth-blocked-reason`，提交按钮 disabled + `aria-busy` 进行中；POST 前 `gitOauthUnboundReasonForRepoUrls` + session `grant_ticket`；taskTaskService `@镜像` 无 `oauth_gitsite` 返回 `git_oauth_comment_grant_missing`）。纯评论不拦截。云端开发层图提交/推送在 git commit 前再校验运行评论 L2，不得先提交再推送失败。
6. 该评论 GET/列表返回相同 `repo_identities`（可为空或不完整）。
7. 克隆进度不在关联项目；评论执行细节仍可显示评论级进度（既有意图）。
8. `@镜像` 后 `comment-composer-git-oauth-hint` 展示已绑定/未绑定/检查中；未绑定文案标明须完成本条评论的 Git OAuth **使用授权**（账号中心已连接时仍需再授权一次，通常秒过），且含真实 `a[href]`，按仓库地址指向 `/api/git-oauth/{github|gitlab}-start-from-gateway/?repo_url=` 且 `grant_kind=pending`（不是账号中心 git-site-oauth）。不得写「仍可发送评论」。
9. 已发出评论的「执行细节」summary 展示该评论 `repo_identities` 对应的 Git 身份标签（与发评选择一致）；无身份则不展示。
10. 同一行展示该评论关联仓库的 Git OAuth 情况（`comment-execution-git-oauth`）：检查中 / 已绑定 / 无写权限 / 未绑定 / **网络不可达**。未绑定须含真实 `a[href]`（`comment-execution-git-oauth-bind`）指向 `/api/git-oauth/{github|gitlab}-start-from-gateway/?repo_url=`。**已绑定但该评论层快照 `git_remote.last_push_error` 为仓库写权限拒绝**（如 GitHub `Permission to … denied`、GitLab not allowed to push、HTTPS 403）时，芯片改为 `data-kind=bound_no_write` 文案「Git OAuth · 无写权限」（红色），并展示「换账号授权」真实 `a[href]`（同一 start-from-gateway）；hover `title` 含完整 push 错误。状态复用任务级已检查的 `repoOAuthReadiness`，摘要行不得再发请求或轮询。任务级检查须在关联项目面板挂载时查询 `user-app-connection`（不得因前端读不到 JS `userId` 而跳过）；OAuth 回流 `?gitlab=ok` / `?github=ok` 后有限次重试再清 query。绑定状态 GET 遇 401 视为未绑定，**不得**整页跳登录（accessCode 分享页 / `skipSessionExpiredRedirect`）。无 OAuth 可授权仓库则不展示。
11. 页面打开且评论区已显示时：若任务级检查判定 AccessToken **处于有效绑定**，须对相关仓触发 **一次** `GET /api/git-oauth/user-app-connection/?repo_url=&probe_access_token=1`（禁止 `setInterval` 轮询）。仅当响应**显式** `access_token_valid: false` **且** `network_status` 不是 `unreachable` 才把摘要改为未绑定并可去绑定；省略 `access_token_valid` 时不得把 `connected: true` 打成未绑定。**非内网 GitLab** 探测时即使 cache 仍有 AccessToken，也须对仓库 origin 做可达性探测；`network_status=unreachable` 时芯片为 `data-kind=unreachable` 文案「Git OAuth · 网络不可达」（不得显示已绑定，也不得改成未绑定/去绑定）。**已标记内网**的 GitLab（`network_status=skipped_intranet`）跳过可达性探测，平台不可达不视为故障，芯片保持已绑定。响应不得含 `access_token`。未绑定不得调用 GitHub/GitLab 换票。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 提交并运行并选定仓库身份 | TASK_COMMENT_IMAGE_MENTIONED | Kafka 既有契约增补 `repo_identities` | taskTaskService handleCreateComment | taskEvents → start-vm | 不新事件名 |
| 展示关联项目 | — | — | — | — | 纯前端只读 |
| 纯评论文本 | — | — | — | — | 无运行副作用 |
| 切换自动克隆子仓 | — | — | — | — | 项目配置 CRUD，无跨服务最终一致需求 |
| 评论/自动运行完成 Git OAuth 打标 | COMMENT_GIT_OAUTH_GRANTED | Kafka | taskGitOauth callback / grant_ticket consume | 审计 | — |
| 评论/自动运行完成 Git OAuth 打标 | COMMENT_GIT_OAUTH_GRANTED | Kafka | taskGitOauth callback / grant_ticket consume | 审计 | — |

## 实施计划

1. 关联项目只读化 + 单测。
2. composer 身份面板 + 草稿；@镜像提示 Git OAuth L1+L2；缺 Git 提交身份不拦截，缺 OAuth L1 或评论 grant_ticket/L2 拦截提交并运行。
3. DDL + comment create/list + snapshot + 事件 payload。
4. 更新引导文案与 Playwright 中「关联项目 OAuth」断言。

## 变更记录

| 日期 | 相对旧版 | 原因 |
|------|----------|------|
| 2026-08-16 | 初版 | 身份生命周期与评论级容器对齐 |
| 2026-08-16 | @镜像提示 Git OAuth，不拦截发评 | 作者未绑 OAuth 时私有仓仍缺凭证（正确失败）；发评不应被挡住 |
| 2026-08-16 | 未绑定入口按仓库跳转 git OAuth start | 不再指向账号中心 `/user/:id/profile/git-site-oauth/` |
| 2026-08-16 | 自动克隆子仓库开关迁到 composer 身份行 | 关联项目只读化时误卸 `TaskDetailNestedReposCloneStatus`，任务详情找不到开关 |
| 2026-08-19 | OAuth 主路径改为创建/编辑任务；composer 为补救 | 自动运行在创建后即克隆，评论区绑定来不及 |
| 2026-08-20 | 容器拉取提交者身份改为评论级 | credential 生产路径仍读任务级表并把作者全部 identity 摊到每仓 |
| 2026-08-21 | 执行细节 summary 回显发评 Git 身份 | 评论已持久化 repo_identities，摘要条未展示导致无法对照 |
| 2026-08-21 | @镜像 未绑 Git OAuth 拦截提交并运行；层图提交并推送在 commit 前再校验 | 评论已创建后推送才换票失败，出现「提交成功但推送失败」 |
| 2026-08-22 | 执行细节 summary 回显 Git OAuth 绑定情况 | 摘要行已有 Git 身份，OAuth 仍只在 composer，已发出评论无法对照 |
| 2026-08-22 | 评论区显示时对有效绑定做一次 AccessToken 探测 | DB `connected` 不代表 refresh 仍可用（gitlab 400），芯片会误显已绑定 |
| 2026-08-22 | 关联项目面板须查询绑定；OAuth 回流重试 | 执行细节「去绑定」跳回后仍显示未绑定：面板从未请求 user-app-connection，且 `gitlab=ok` 未消费 |
| 2026-08-22 | 绑定查询 401 不踢登录 | 挂载即查后 accessCode 页被 forward-auth 401 整页踢到登录 |
| 2026-08-22 | probe 省略 `access_token_valid` 不得打成未绑定 | 公网 probe 200 且 `connected: true` 但不带该字段，芯片被误标未绑定 |
| 2026-08-29 | 已绑定 + 仓库写权限拒绝：OAuth 芯片 overlay「无写权限」+ 换账号授权；zTree 标签「push 无权限」 | 绿「已绑定」与红「push 失败」（Permission denied）并存，用户无法分辨是未绑定还是账号对该仓无写权限 |
| 2026-08-29 | 评论 JSON 增 oauth_* L2；自动运行走 grant_ticket | ADR-0049：无 L2 不得换票 |
| 2026-08-29 | 「提交并运行」与云端推送在创建评论 / git commit 前校验评论 L2；task GET 附 `comment_oauth_grants` | 仅校验 L1 时评论能创建、提交成功后推送才报尚未完成使用授权 |
| 2026-08-29 | 非内网 GitLab OAuth 探测不可达 → 芯片「网络不可达」；内网跳过 | cache hit 仍显示已绑定；内网超时被误标未绑定 |
