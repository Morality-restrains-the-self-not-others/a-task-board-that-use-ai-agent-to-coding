# 意图：创建/Fork 任务时完成仓库 Git OAuth 绑定（自动运行）

- **日期**: 2026-08-19
- **状态**: 已实施

## 背景与目标

任务详情引导克隆失败时曾提示「请在下方添加评论时完成 OAuth 绑定」。容器在创建/Fork 任务勾选**自动运行**后就会克隆，评论区绑定来不及。OAuth 是用户级连接，应在**创建/编辑任务**或 **Fork 并自动运行**前由操作者绑齐。

## 范围与边界

- 范围内：创建/编辑任务弹窗在 **auto_run=true** 时对所选项目中 GitHub/GitLab 仓检查 **本会话 grant_ticket 或该仓所属项目对当前用户该 gitsite 的项目 L2**（`validate-git-repos` + `project_id` 为 `token_available`）。**不是**仅 L1 `user-app-connection.connected`。未齐则禁用提交；每仓真实 `<a href>` 指向 git-oauth start（`grant_kind=pending`）。项目详情已授权（同用户同项目同站）视为已绑定。
- 范围内：**Fork 确认弹窗**在用户选择「自动运行并派生」后展示 OAuth 状态；未齐则禁用「确认派生」并给出绑定链接。
- 范围内：引导克隆失败文案指向创建/编辑任务；评论 composer 的 OAuth 提示为补救路径。`@镜像` 提交并运行在未绑定 GitHub/GitLab OAuth 时拦截；纯评论不拦截。
- 范围内：克隆凭证 `ResolveProvider` 必须命中区域 GitLab 实例键（插值 `website`）；禁止 `gitlab:default` 误回退。YAML 未命中的租户 Path A IP GitLab（如 `http://115.29.110.74/...`）须用 `gitsite:{host}` 走 gitOauth `/api/internal/gitsite/{host}/oauth/access-for-user/`（与 fork「已绑定 Git OAuth」同一套 `ResolveProviderByGitsiteWithDB`），不得把已有 identity 的仓标成 `REPO_CLONE_CREDENTIALS_INCOMPLETE`。`user-app-connection` 在 `repo_url` 能匹配实例 website 时只查该实例键。Git/OAuth 软跳过不得创建【自动运行】评论、不得 start-vm。
- 范围外：不把 Git 提交身份从评论级改回任务级；**未勾选自动运行**时不拦截创建/Fork（OAuth 可在后续评论运行前补救）。不恢复跨 GitLab 实例 prefix fallback。OAuth 成功路径发布 `COMMENT_GIT_OAUTH_GRANTED` / `PROJECT_GIT_OAUTH_GRANTED`（审计）。

## 约束与风险

- 无 GitHub/GitLab 仓（或 generic Git）不拦截。
- 检查中 / 检查失败均禁用提交（失败须 `data-traceId`）。
- 绑定入口必须是真实 `href`，禁止 `@click` 冒充链接。

## 验收标准

1. 创建任务 **auto_run=true** 且所选项目含需 OAuth 的仓库，且当前用户**既无**本会话 grant_ticket **也无**该项目该 gitsite 的项目 L2：提交按钮 disabled，出现 `create-task-submit-blocked-reason`（文案含「OAuth」），每仓出现 `a[data-testid=create-task-repo-oauth-bind]`。仅 L1 connected、无项目 L2、无 ticket **仍拦截**。项目详情已对该仓打上项目 L2（`token_available`）则 **不拦截**，出现 `create-task-repo-oauth-bound`。
2. 创建任务 **auto_run=false**：不因 OAuth 拦截提交；不展示 OAuth 绑定行。
3. Fork 弹窗含 OAuth 仓且未绑定：选择「自动运行并派生」后 `fork-confirm-submit` disabled，出现 `fork-auto-run-oauth-bind`；切回「不自动运行，仅派生」后可确认。
4. 全部已绑定：无拦截；仓库行出现 `create-task-repo-oauth-bound` 或 `fork-auto-run-oauth-bound`。
5. `formatContainerBootstrapFailureHint` 在凭证不齐时含「创建或编辑任务」，**不含**「添加评论」。
6. composer `comment-composer-git-oauth-hint` 未绑定文案含「创建或编辑任务」，**不含**「仍可发送评论」。纯评论不因缺 OAuth 拦截；`@镜像` 提交并运行未绑定时拦截（按钮 disabled + POST 前再校验）。
7. `https://gitlab-tencent-sh-1.*` 解析为 `gitlab:tencent-sh-1`，未匹配 GitLab host 不得落到 `gitlab:default`。Path A IP 仓（`http://115.29.110.74/...`）在 YAML miss 且任务已有 git identity 时，`BuildRepoCloneCredentials` 须换票成功而不是 `missing_repo_credentials`。
8. 仅绑 `gitlab:tencent-sh-1` 时，`user-app-connection?repo_url=` 指向另一 GitLab host → `connected:false`。
9. `StartSkipReason` 非空时不 `ensureAutoRunAtComment`、不 start-vm；跳过原因仍落库供详情横幅。
10. 引导克隆失败错误节点 `[data-testid=comment-layer-ztree-loading-error]` 在 SSE 带 `trace_id` 时有 `data-traceId`，无则省略。
11. 层图仅有 `meta_kind=empty` / `bootstrap_pending` 锚点时，凭证失败仍展示 `comment-layer-ztree-loading-error`（含「Git 授权未齐」）于 **`comment-layer-ztree-panel` 内**（不拆掉该面板）；层图 SSE 不得因此清掉 `BOOTSTRAP_FAILED` 文案。无树节点时仍用 overlay。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 创建任务前检查/引导 OAuth | COMMENT_GIT_OAUTH_GRANTED / PROJECT_GIT_OAUTH_GRANTED | Kafka | 回调 MarkGrant / 消费 ticket / 项目 L2 seed / Fork 源评论 L2 seed | 审计；自动运行评论换票 | — |

## 路径分片键 / 幂等（NFR 摘要）

- 创建任务路由已含 `tenantId` / `workspaceId`；OAuth 连接按当前用户 + `repo_url` 查询，键粒度合适。
- 写路径为既有 OAuth start（用户级绑定，重复授权幂等）；本增量无新 POST。

## 实施计划

1. 纯函数门禁 + 单测。
2. 创建弹窗接线 + 真实绑定链接。
3. 失败提示与 composer 补救文案。
4. 更新 034 与价值流测试点。

## 变更记录

| 日期 | 相对旧版 | 原因 |
|------|----------|------|
| 2026-08-19 | 初版 | 创建者应在创建任务时完成 OAuth，而非评论时 |
| 2026-08-19 | 后端克隆键与门禁按实例 | 区域 GitLab website 未插值 → `gitlab:default`；连接检查展开全部 GitLab 键误放行；软跳过仍建自动运行评论 |
| 2026-08-21 | @镜像 提交并运行未绑 OAuth 改为拦截 | 评论创建后层图推送才换票失败，用户看到「提交成功但推送失败」 |
| 2026-08-27 | 空层锚点不得挡住凭证失败 overlay | 心跳空层 `bootstrap_pending` 使 nodeCount>0 且层图 SSE 清掉失败文案，任务关联永远「正在准备可写层」 |
| 2026-08-29 | 门禁改为本会话 grant_ticket，不看项目 L2 / 不仅看 L1 connected | ADR-0049 资源使用标记 |
| 2026-09-02 | 创建任务默认选中最近打开的项目详情；OAuth 行 URL 与校验结果按 clone key 对齐 | 工作面板默认 projects[0] 与详情已授权项目不一致，弹窗仍显示「OAuth 绑定」 |
| 2026-09-02 | Fork 自动运行评论第三种 L2：源任务同用户评论（`via=fork_source_comment_l2_seed`） | 源任务 ticket 已消费且无项目 L2 时新评论 BINDING_MISSING |
