# 意图：PR 链接回复、合并状态与一键合并

- 日期：2026-08-22
- 状态：implementing
- 相关页面：任务详情评论会话 / zTree PR 按钮

## 背景与目标

推送生成 PR 后，用户希望在触发评论下看到回复（PR 链接）、合并状态，以及可审计的一键合并。现有 PR 按钮只外链审查页；「合并到目标分支」不是远端 MR merge。

## 范围与边界

**范围内：**

- 推送返回 `html_url` 后创建人类回复评论：`POST /api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent_comment_id}/`（`git_pr` 在 body，parent 在 path）
- 回复展示链接、open/merged/closed、未合并时一键合并
- GitOauth 状态/合并 API + 审计
- 同任务同 URL 幂等

**范围外：**

- 容器本地 `container-layer-git-merge`
- 后台轮询 MR 状态
- 站内完整 PR 审查 UI

## 约束与风险

- 无 Git OAuth 绑定时不能查/合；须明确错误
- Git 平台拒绝合并（冲突/权限）须展示错误 + `data-traceId`
- 创建回复失败不得让用户以为 push 失败

## 验收标准

1. 有 `html_url` 的推送后，父评论下出现回复，正文/卡片含该 URL。
2. 重复同一 URL 不新增第二条。
3. 加载评论后可见合并状态；已合并无「一键合并」。
4. 有权限时一键合并成功，状态变为已合并；审计含点击者 user_id。
5. 无后台轮询。
6. GitLab API origin 必须使用 html_url 的 scheme/port，或租户 Path A `base_url` / provider `website`（禁止把 HTTP GitLab 默认成 `https://host:443`）。

## 实施计划

见 `docs/superpowers/plans/2026-08-22-pr-reply-merge-status-plan.md`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | MQ类型/契约 |
|---------|----------------|--------|--------------|------------|
| 记录 PR 回复评论 | TASK_GIT_PULL_REQUEST_RECORDED | taskTaskService handleCreateComment（首次 git_pr 插入） | 同次发布 SSE_MESSAGE(`task_git_pr_reply_created`) → sse_message/1_send_sse_message → 任务详情重拉 Feed | Kafka topic `task-git-pull-request-recorded` + `sse-message` |
| 一键合并远端 PR | GIT_MERGE_REQUEST_MERGED | taskGitOauth merge-request-merge（远端成功） | 本期无消费者；`git_oauth_taskcredentialaudit` | Kafka topic `git-merge-request-merged` |

## 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-08-27 | GitLab MR 状态/合并 API 保留 http(s)+port；优先 provider website / 租户 BaseURL | `Origin()` 写死 `https://` 导致 HTTP GitLab（如 `http://115.29.110.74`）被打到 443 connection refused |

## 实施计划

见 `docs/superpowers/plans/2026-08-22-pr-reply-merge-status-plan.md`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | MQ类型/契约 |
|---------|----------------|--------|--------------|------------|
| 记录 PR 回复评论 | TASK_GIT_PULL_REQUEST_RECORDED | taskTaskService handleCreateComment（首次 git_pr 插入） | 同次发布 SSE_MESSAGE(`task_git_pr_reply_created`) → sse_message/1_send_sse_message → 任务详情重拉 Feed | Kafka topic `task-git-pull-request-recorded` + `sse-message` |
| 一键合并远端 PR | GIT_MERGE_REQUEST_MERGED | taskGitOauth merge-request-merge（远端成功） | 本期无消费者；`git_oauth_taskcredentialaudit` | Kafka topic `git-merge-request-merged` |
