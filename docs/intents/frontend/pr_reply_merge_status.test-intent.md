# 测试意图：PR 链接回复、合并状态与一键合并

- 对应功能意图：`pr_reply_merge_status.intent.md`
- 日期：2026-08-22

## 测试目标

证明：PR 创建后有幂等回复；状态正确；一键合并走 GitOauth 并审计；无轮询。

## 测试分层

| 层 | 覆盖 |
|----|------|
| 单元 Go | URL 解析、评论幂等、事件发布、merge handler 审计 |
| 单元 JS | 推送后 POST 回复、嵌套 parent、卡片状态/按钮、防重放 |
| 不测 | 真实 GitLab 公网 merge（用 httptest mock） |

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| U1 | 推送 html_url | POST `/api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent}/`，body 含 git_pr.html_url，无 body parent_comment_id |
| U2 | 同 URL 再推 | 不第二次 POST 或服务端返回已有 id |
| U3 | nest | 带 parent 的人类评论出现在 children |
| U4 | 解析 GitLab URL | host/project/iid |
| U5 | 解析 GitHub URL | owner/repo/number |
| U6 | status opened | 徽章「未合并」，显示一键合并 |
| U7 | status merged | 徽章「已合并」，无合并按钮 |
| U8 | merge 200 | 调 Git API merge；审计 action=merge_request_merge |
| U9 | merge 无 token | 401/403 友好错误 |
| U10 | 事件 | 首次评论发 TASK_GIT_PULL_REQUEST_RECORDED + SSE_MESSAGE(task_git_pr_reply_created)；merge 成功发 GIT_MERGE_REQUEST_MERGED |
| U11 | HTTP GitLab html_url | 状态 API 使用 `http://host`，不得默认 443 |
| U12 | html_url 为 https 但 website 为 http | 状态 API 使用 provider website 的 http origin |
| U13 | SSE 推送 | 打开任务详情时收到 task_git_pr_reply_created 后评论 Feed 出现 git_pr 气泡，无需整页刷新 |

## 数据与环境

- 测试 SQLite/httptest；不连真实 Git
- 前端 vitest mock apiFetch

## 通过标准

上表全部绿；无 setInterval 拉 status。
