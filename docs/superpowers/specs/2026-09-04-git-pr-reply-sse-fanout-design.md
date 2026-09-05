# Design: Git PR 回复评论经 SSE 推送到任务详情

- **Date:** 2026-09-04
- **Status:** accepted (goal-mode auto)
- **Iteration:** git-pr-reply-sse-fanout

## Problem

任务详情页在未整页刷新时，服务端异步创建的 `git_pr` 子评论（层图回填 / auto-run / LayerPush POST）不会出现在会话 Feed。已有 OPT-20260903-002 补偿重拉在部分路径仍会漏；`TASK_GIT_PULL_REQUEST_RECORDED` 已发布但意图文档写明「本期无消费者」。

## Decision

**We will** 在 `handleCreateComment` 首次插入 `git_pr` 评论成功后，除既有 `TASK_GIT_PULL_REQUEST_RECORDED` 外，再发布 `SSE_MESSAGE`，`status_data.event_name = task_git_pr_reply_created`，经既有 `sse_message/1_send_sse_message` → Redis `sse:{task_id}` → taskSSE → 任务详情 EventSource。前端收到后幂等触发评论 Feed 重拉（`fetchTaskDetail`）。

不新建独立 Kafka intent 消费者：复用既有 SSE 投递链，降低运维面；领域事件仍保留供审计/后续扩展。

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 新建 `task_git_pull_request_recorded/1_fanout_sse` intent | 多一跳 Kafka + 新 runAll 进程；本期载荷已够 FE 重拉 |
| 仅依赖层图补偿重拉 | 已存在仍漏；不覆盖非层图创建路径 |
| SSE 内嵌完整评论 JSON | 需作者头像/合并状态 enrichment；重拉更稳 |

## Contract

```json
{
  "task_id": "<taskId>",
  "status_data": {
    "event_name": "task_git_pr_reply_created",
    "comment_id": "cmt_…",
    "parent_comment_id": "cmt_…",
    "git_pr_html_url": "https://…/merge_requests/N",
    "git_pr_provider": "gitlab|github|",
    "tenant_id": "…",
    "workspace_id": "…",
    "created_by_user_id": "…",
    "message": "PR 回复评论已创建",
    "trace_id": "…"
  }
}
```

幂等：同 `task_id`+`git_pr_html_url` 已存在时不插入、不发事件（既有行为）。

## FE

`establishSSEConnection` 识别 `task_git_pr_reply_created` → 调用注入的 `onTaskGitPrReplyCreated`（默认 `fetchTaskDetail`）。同 comment_id 去重，避免重复重拉。

## Architecture

见 `docs/architecture/v131-*`（application-integration）。

## NFR / Permissions (compressed)

- 路径分片：事件与 SSE 均按 `task_id`（已有）
- 幂等：L2 — 创建侧 URL 唯一；FE 按 `comment_id` 去重
- 权限：无新 API；SSE 仍走既有任务详情订阅鉴权
- 无进程内轮询
