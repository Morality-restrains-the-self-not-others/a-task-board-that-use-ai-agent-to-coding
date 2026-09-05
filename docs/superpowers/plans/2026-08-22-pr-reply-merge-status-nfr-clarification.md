# NFR 澄清：PR 回复与一键合并

- **日期:** 2026-08-22
- **价值流:** `docs/superpowers/plans/2026-08-22-pr-reply-merge-status-value-stream.md`

默认 L2；资金/云资源路径未涉及，合并为 Git 写路径按 L3 幂等。

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 可伸缩性 | 动作 |
|------|---------|----------|----------|------|
| POST `/api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent_comment_id}` | tenant_id + workspace_id + task_id | 是（任务表按 tenant/workspace/task） | L1 | 回复专用；parent 在 path |
| POST `/api/tasks/{taskId}/comments/tenant_id/{tid}` | tenant_id + task_id | 是（任务表按 task/tenant） | L1 | 顶层发评保留 |
| POST `/api/git-oauth/merge-request-status/tenant_id/{tid}` | tenant_id | 是（租户成员校验） | L1 | html_urls 批量上限 20 |
| POST `/api/git-oauth/merge-request-merge/tenant_id/{tid}` | tenant_id | 是 | L1 | 业务键 html_url |
| 前端路由 task-detail | tenantId + workspaceId + taskId | 是 | L0 页面 | — |
| Kafka TASK_GIT_PULL_REQUEST_RECORDED | task_id key | 是 | L1 | 生产投递 |
| Kafka GIT_MERGE_REQUEST_MERGED | html_url 或 task_id | 用 task_id 避免热点 URL | L1 | key=task_id |

无「无分片 ID 却声称无需伸缩」的路径。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| 创建 git_pr 评论 | 插入评论 | 双击推送、刷新重试、关页重推 | 同一 task 同一 PR URL | `(task_id, git_pr_html_url)` | 返回已有评论 |
| merge-request-merge | 远端 merge + 审计 | 双击、超时重试 | 同一 html_url 同一用户意图 | Idempotency-Key + 远端已 merged 视为成功 | 已合并 → 200 merged，再写审计注明 noop |
| merge-request-status | 无（只读） | — | — | L0 | — |
| 事件消费 | 本期无消费者 | — | — | 有消费者时键=评论 id / merge 审计 id | — |

资金/配额：不适用。云资源：不适用。Git merge 视为 L3 外部写。

## 支撑程度

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L2 | 批量 status ≤20；评论按 task 查询 |
| 数据一致性 | L2 | Git 为 SSOT；评论幂等 |
| 安全 | L3 | SSRF allowlist host；审计 PII 仅 user_id |
| 可用性 | L2 | Git 超时不拖垮任务详情（status 独立请求） |
| 可观测性 | L2 | 结构化 event 名 |
| 容错 | L2 | 建评失败不影响 push 成功提示 |

## 质量场景

1. **刺激：** 连续两次 push 同一 MR。**响应：** 仅一条回复。
2. **刺激：** 已 merged 再点一键合并。**响应：** 200，状态已合并，审计 noop。
3. **刺激：** html_url 指向未配置 host。**响应：** 400，不发 HTTP。

## 领域模型影响

- 值对象 `MergeRequestRef`（provider, host, project, iid）
- 聚合：TaskComment（git_pr 属性）；MergeAttempt 以审计记录表达，不另建模聚合
