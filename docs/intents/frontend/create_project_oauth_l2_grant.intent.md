# 功能意图：创建项目 OAuth 回流后写入项目 L2

## 用户故事

作为租户成员，我在**创建项目**时已对 GitLab/GitHub 仓库完成 OAuth 授权，打开项目详情「云端开发」Git 仓库行应显示「已授权」，而不是「需要授权」。

## 背景

ADR-0049 将 Git OAuth 分成 L1（taskGitOauth 用户凭证）与 L2（taskProjectService `project_git_oauth_grant`）。详情页 `validate-git-repos` 带 `project_id`：仅有 L1、没有项目 L2 时状态为 `not_bound`，UI 文案为「需要授权」。

创建项目时尚无 `project_id`，OAuth 回调使用 `grant_kind=pending` 并签发一次性 `grant_ticket`。创建任务路径会消费 ticket 写成评论 L2；创建项目路径原先既不提交也不消费 ticket，因此新建项目永远没有 L2。

## 验收标准

1. 创建页 OAuth 回流后，`grant_ticket` 按仓库 gitsite 记入 sessionStorage。
2. 创建项目 POST 携带 `grant_ticket` / `grant_tickets`。
3. taskProjectService 对每个仓库 gitsite 调用 gitOauth `grant-ticket/consume/`，成功则 upsert `project_git_oauth_grant` 并发布 `PROJECT_GIT_OAUTH_GRANTED`（`via=grant_ticket`）。
4. 消费失败不阻断创建（仍 201，无 L2；用户可在详情页再授权）。
5. 存量项目（创建于本修复之前）不自动回填 L2；须在详情页点一次「OAuth 授权」（`grant_kind=project`）。

## 范围

- taskFE `grantTicketSession` / `useCreateProjectGitRepoRows` / `useCreateProjectForm`
- taskProjectService `applyCreateProjectGrantTickets`
- 不改 ADR-0049 门禁语义（详情仍看项目 L2，不把 L1 当成已授权）

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 例外理由 |
|---------|--------|--------|----------|
| 创建项目消费 pending ticket 写成项目 L2 | PROJECT_GIT_OAUTH_GRANTED | 创建 POST 成功后 consume | 与详情页 MarkGrant 同事件；`via=grant_ticket` |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-09-02 | 初版 |
