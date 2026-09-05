# PR 链接回复、合并状态与一键合并

- **日期:** 2026-08-22
- **状态:** accepted（/goal 自动采用）
- **作者:** cursor
- **ADR:** [ADR-0028](../../adr/0028-pr-reply-one-click-merge-audit.md)
- **页面:** 任务详情评论会话（zTree「PR」按钮所在评论卡）

## 目标

生成 GitHub PR / GitLab MR 链接后：

1. 在触发该推送的评论下自动创建一条**回复**，正文为 PR 链接。
2. 回复上展示该 PR **是否已合并**（open / merged / closed）。
3. 未合并时提供 **一键合并**。
4. 记录**谁点击了一键合并**（审计）。

成功标准：

- 推送返回 `html_url` 后，会话中出现嵌套回复，含可点击 PR URL。
- 同一 `(task_id, html_url)` 重复推送不产生第二条回复。
- 打开任务详情时（用户触发加载）刷新合并状态；无后台轮询。
- 点击一键合并后 GitLab/GitHub 执行 merge；审计表写入 `user_id`、`html_url`、结果。
- 已合并则隐藏合并按钮，展示「已合并」。

## 🕸️ Code Review Graph 分析

`CRG: code-review-graph update --brief` 成功（软依赖）。本会话无 codegraph MCP；按源码调用链设计。

既有链：

- zTree `layer-ztree-pr-btn` ← `prHtmlUrl` ← `git_remote.pr_html_url` ← `onLayerGraphLayerPush` 读 `github_pull_request.html_url`
- 会话 `TaskDetailConversationFeed` 用 `parent_comment_id` **仅**嵌套 `container_agent`
- 人类评论 `POST /api/tasks/{taskId}/comments/tenant_id/{tid}` → `task_comments`（无 parent / git_pr 列）
- 合并按钮「合并到目标分支」是**容器本地 git merge**，不是合并远端 PR
- `taskGitOauth` 持有用户 token + `git_oauth_appaccesstokenuseaudit` / `git_oauth_taskcredentialaudit`

## 当前架构理解

- 业务层：租户成员在任务评论中推送并审查 PR
- 应用层：taskFE、taskTaskService、taskGitOauth、taskContainerGateway（已有 PR 创建）
- 技术层：MySQL `task_task` / `git_oauth`；GitHub/GitLab 公网 API
- 当前基线：v93 current

本次在此基础上扩展评论与 OAuth 写路径，不新建服务。

## 决策（锁定）

### 1. 回复落点：人类评论 + parent_comment_id

推送成功且有 `html_url` 时，前端以当前用户身份 `POST`：

`/api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent_comment_id}/`

```json
{
  "content": "https://gitlab-tencent-sh-1.daydaymoney.com/group/repo/-/merge_requests/12",
  "git_pr": { "html_url": "...", "provider": "gitlab" },
  "execution_mode": "independent"
}
```

- `execution_mode=independent`：不触发 @镜像 Agent。
- **parent 必须在 path**；无执行评论 id 或无 workspace 时跳过建评（不打顶层评论接口）。
- `nestDisplayCommentsByParent` 改为：**任意**带有效 `parent_comment_id` 的评论都嵌套（不再仅限 container_agent）。

### 2. 幂等

`task_comments` 增加 `parent_comment_id`、`git_pr_html_url`、`git_pr_json`。

创建时若同 `task_id` + 非空 `git_pr_html_url` 已存在 → 返回已有评论（200），不插第二行。

### 3. 状态与合并：taskGitOauth（token owner）

| 方法 | 路径 | 作用 |
|------|------|------|
| POST | `/api/git-oauth/merge-request-status/tenant_id/{tid}` | body `{html_urls:[]}` 批量查 state |
| POST | `/api/git-oauth/merge-request-merge/tenant_id/{tid}` | body `{html_url, task_id, comment_id}` 一键合并 |

- 鉴权：会话用户 + `ensureTenantMember`。
- Token：当前用户对该 git host 的 OAuth（与推送同一套）。
- Git 侧 ACL 仍由 GitHub/GitLab 执行（无权限 → 403，文案不泄漏 token）。
- 状态 SSOT 在 Git 平台；库内不缓存 merged 作为真源。前端在评论加载/合并成功后拉取，**禁止 setInterval 轮询**。

### 4. 审计

一键合并（无论成功失败）写入：

1. `git_oauth_taskcredentialaudit`：`action=merge_request_merge`，`task2app_user_id`、`task_id`、`html_url`、`state`、`ok`
2. `git_oauth_appaccesstokenuseaudit`：token 指纹（既有 token 使用审计）

### 5. 领域事件

| 意图 | 事件 | 发布点 |
|------|------|--------|
| 记录 PR 回复评论 | `TASK_GIT_PULL_REQUEST_RECORDED` | taskTaskService 首次插入 git_pr 评论后 |
| 一键合并被接受 | `GIT_MERGE_REQUEST_MERGED` | taskGitOauth merge API 远端成功后 |

本期无强制消费者（SSE/审计以表为准）；事件用于对照与后续投影。

## 方案对比

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| A. 仅 zTree 上展示状态/合并 | 改动小 | 用户明确要求「回复」；评论时间线看不到 | 拒绝 |
| B. 网关 PR 创建后 Kafka 建评 | 关页也不丢 | 网关无会话用户、comment_id 常缺失 | 拒绝作主路径 |
| C. 前端建评 + GitOauth 状态/合并 + 评论幂等 | UX 即时；作者即点击人；审计完整 | 关页过早可能漏评（幂等可补点） | **采用** |

## 不做

- 不把「合并到目标分支」（容器本地 merge）改成远端 PR merge
- 不在评论列表 GET 里同步打 Git API（N+1 / 超时绑死任务详情）
- 不后台轮询 MR 状态
- 不新建服务；不落 Django

## Python 新增接口清单

无。

## 架构变更

需要 v94 application-integration + enterprise-landscape（四件套）。
