# 意图：任务详情打开时 Git OAuth AccessToken 探测失败以内联徽标展示

- **日期**: 2026-09-02
- **状态**: 已实施

## 背景与目标

打开任务详情时，评论区会对「看起来已绑定」的仓库做一次 Git OAuth AccessToken 探测（`GET /api/git-oauth/user-app-connection/?probe_access_token=1`）。前端 6s 超时 abort 后，原先经 `showRequestError` 弹出全屏错误模态，挡住页面。探测失败（超时 / 服务不可用）应落在评论执行细节的 Git OAuth 徽标上，而不是弹窗。

## 范围与边界

- 范围内：`useCommentGitOauthAccessTokenProbe` 失败路径不再调用 `showRequestError`；`CommentExecutionGitOauthBadge` 增加 `check_failed` 态（超时文案「Git OAuth · 检查超时」），芯片挂 `data-traceId`。
- 范围内：探测函数按仓捕获超时，返回 `checkFailedRepoUrls` 而非向上抛。
- 范围外：不拉长前端 6s 超时；不改为轮询重试；用户主动推送等写路径仍可用错误弹窗。GitHub `issue_failed` 探测耗时（约 15s）属后端 OPT。

## 约束与风险

- 错误 UI 必须带 `data-traceId`（元规则 24）；无 trace 则省略。
- 禁止进程内轮询；同一 URL 仍只探测一次。
- 检查失败不得误显示「已绑定」。

## 验收标准

1. 探测超时：页面不出现 `.app-modal-overlay` 错误弹窗。
2. 评论执行细节出现 `data-testid=comment-execution-git-oauth` 且 `data-kind=check_failed`，文案含「检查超时」。
3. 有 traceId 时芯片带 `data-traceId`。
4. 纯评论浏览不被弹窗打断。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 例外理由 |
|---------|--------|---------|
| 展示探测失败徽标 | — | 纯前端只读展示，无领域状态变更 |

## 变更记录

- 2026-09-02：初版。traceId `e0a7ec3d-133a-4ff9-aa9d-15780e06d1eb`：gateway 499 @6s，taskGitOauth 200 @14.8s，`git_oauth_access_token_probe valid=false reason=issue_failed`。
