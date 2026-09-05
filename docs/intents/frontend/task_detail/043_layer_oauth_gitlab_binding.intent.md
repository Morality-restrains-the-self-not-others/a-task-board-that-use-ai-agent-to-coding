# 意图：层图 push 换票须识别 GitLab 评论 L2（勿误报 BINDING_MISSING）

- **日期**: 2026-09-02
- **状态**: 已实施
- **相关页面**: 任务详情云端开发 zTree 层节点（`layer-ztree-push-error-label`）

## 背景与目标

任务 `task_882908895993950208` 层节点红色「push 失败」：容器 `layer-github-oauth-access-tokens` 返回 HTTP 409 `BINDING_MISSING`，`缺少绑定: gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work`，`github_auth_by_repo: {}`。

根因不是用户没授权，而是换票链路丢掉了评论级 GitLab L2：

1. 评论 `repo_identities_json` 允许站点级授权行（`repo_url` 空、`oauth_gitsite` 有值，见 `handleInternalMarkCommentGitOAuthGrant`）。
2. credential `FetchRepoIdentities`（SQLite / HTTP）丢弃空 `repo_url` 行。
3. `anyCommentSelectedIdentity` 只认 `git_identity_id`，站点级 L2 被作者 Git 身份覆盖。
4. 层 OAuth 按仓库 **精确 URL** 查身份，HTTPS / SSH / `.git` 后缀不一致即当作未绑定。
5. 失败文案指向只读的「关联项目」，且 `detail_safe` 写「GitHub」。

目标：已打评论 L2（含站点级）的 GitLab 仓换票成功；真缺绑定时芯片为「未绑定 Git 授权」，并指向创建/编辑任务或评论「提交并运行」的授权入口。

## 范围与边界

- 范围内：`taskCredentialService` 身份解析与换票；zTree 失败芯片文案；容器 refresh-push 错误前缀「Git OAuth」而非「GitHub」。
- 范围外：不绕过 ADR-0049（评论无 `oauth_gitsite` 仍不得换票）；不在关联项目面板恢复 OAuth 编辑控件。

## 验收标准

1. 评论 JSON 仅有 `{repo_url:"", oauth_gitsite:"gitlab-tencent-sh-1.daydaymoney.com"}` 时，`FetchRepoIdentities` 仍返回该行（FromComment）。
2. 层 OAuth：任务仓 URL 与身份 URL 仅 scheme/`.git`/scp 不同、或仅有站点级 L2 + 评论作者 `user_id` 时，按 match key 换票成功，`git_auth_by_repo_match_key` 含 `gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work`。
3. 真无绑定时 `error_code=BINDING_MISSING`；`detail` 含「创建或编辑任务」或「提交并运行」，**不含**「关联项目」；`detail_safe` 含「Git 授权」，**不含**单独「GitHub」。
4. zTree：`last_push_error` 含 `BINDING_MISSING` /「缺少绑定」时 `pushErrorLabel==='未绑定 Git 授权'`，`pushErrorKind==='binding'`。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 例外理由 |
|---------|--------|---------|
| 换票与失败展示 | — | 读已有 `COMMENT_GIT_OAUTH_GRANTED`；本增量不新发布事件 |

## 变更记录

| 日期 | 相对旧版 | 原因 |
|------|----------|------|
| 2026-09-02 | 初版 | 公网 GitLab ram-work 层图 push 409 BINDING_MISSING |
