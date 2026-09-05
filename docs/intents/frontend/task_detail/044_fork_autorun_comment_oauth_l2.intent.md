# 意图：Fork 自动运行评论继承源任务同用户 Git OAuth L2

- **日期**: 2026-09-02
- **状态**: 已实施
- **相关页面**: 任务详情云端开发 zTree（Fork 派生的【自动运行】评论）

## 背景与目标

Fork 并自动运行时，新【自动运行】评论只带 Git 提交身份、不带 `oauth_gitsite`。源任务评论上用户已完成 Git 授权，但 grant_ticket 已消费、项目 L2 可能缺失，层图 push 报 HTTP 409 `BINDING_MISSING`。

目标：同一操作者 Fork 出的自动运行评论，按 gitsite 种下源任务评论 L2；存量评论在 container-snapshot 读取时补种。

## 范围与边界

- 范围内：`taskTaskService` `prepareAutoRunRepoIdentities` 第三源；snapshot 缺 L2 时 UPDATE。
- 范围外：不绕过 ADR-0049；不把他人评论 L2 拷到新评论；不改 credential 换票协议。

## 验收标准

1. 源任务同用户评论有 `{oauth_gitsite: gitlab-tencent-sh-1.daydaymoney.com, oauth_remote_user_id}`，Fork 任务 `fork_from` 指向源、新评论仅 `git_identity_id` → 种下后含该 gitsite。
2. 源评论作者 ≠ Fork 操作者 → 不种下。
3. 任务无 `fork_from` → 不种下。
4. 新评论已有 `oauth_gitsite` → 不覆盖。
5. `COMMENT_GIT_OAUTH_GRANTED` payload `via=fork_source_comment_l2_seed`，幂等键 `grant:comment:{commentID}:{userID}:{gitsite}`。
6. snapshot GET 对缺 L2 的 Fork 自动运行评论补种并写回 `repo_identities_json`。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 例外 |
|---------|--------|--------|------|
| Fork 自动运行评论种下 L2 | COMMENT_GIT_OAUTH_GRANTED | `applyForkSourceCommentL2SeedToIdentities` | — |

## 变更记录

| 日期 | 相对旧版 | 原因 |
|------|----------|------|
| 2026-09-02 | 初版 | 公网 Fork 自动运行评论 ram-work 层图 push 409 BINDING_MISSING |
| 2026-09-02 | snapshot 在项目 L2 GET 失败时仍按仓库 host 补 `oauth_gitsite`；沿 fork_from 链取祖父评论 L2 | 现网 task-project-service GET grant 405 导致种下失败；中间 Fork 评论也无 L2 |
