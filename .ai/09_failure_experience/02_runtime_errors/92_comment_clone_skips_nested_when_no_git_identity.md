# [运行时] 任务级自动克隆已开，评论级项目克隆只有元仓

## 现象

任务详情「自动克隆子仓库」打开，且「子仓库克隆状态」显示 `34/34 已完成`；同一评论「执行细节」里项目克隆只有 `项目克隆 (1/1) 完成 ram-work`。

- **页面**：任务详情 → 关联项目 + 评论执行细节
- **task**：`task_876469535748681728`
- **启动 TraceId**：`d8b9efda4347cba2de274d64`

## 日志时间线（本地 logs，Loki job 标签为空）

| 时间 | 服务 | 事件 |
|------|------|------|
| 23:00:06 | task-credential-service | `repo identities fetched count=0` |
| 23:00:06 | task-credential-service | `task repos fetched projects=1 repos=1 auto_clone_nested_off=0` |
| 23:00:06 | task-credential-service | `POST .../repo-clone-credentials/` **409** `credentials built total=1 ok=0 missing=1` |
| 23:01:41 | task-credential-service | `POST .../task-detail/` **200** duration_ms=5（未调用 nested-git-repos，该 API 通常 2–4s） |
| 23:15:40 | taskProjectService | 前端 JWT 用户发现 nested-git-repos **200**（34 个子仓） |

## 根因

1. **评论/容器 enrich 依赖 `task_repo_identities` 的 userID**。`MergeNestedReposIntoSnapshots` 与 `NestedGitReposHTTPClient` 在 `userID<=0` 时直接跳过发现。本任务 identities=0 → 克隆任务列表只有父仓 ram-work → 日志 `(1/1)`。
2. **任务级 34/34 是假完成**：`resolveNestedRepoCloneStatus` 在无进度行时若 `bootstrapCloneDone && containerReady` 把发现列表全部标成已完成，即使这些子仓从未进入 bootstrap 克隆任务。
3. 前端 nested 发现走登录用户 JWT，与容器 token 路径脱节，所以 UI 能列出 34 个子仓，容器却只克隆 1 个。

## 修复

1. `userID=0` 仍调用 `/api/internal/nested-git-repos/`（省略 `user_id`），GitHub 公开仓走匿名 Contents API。
2. 去掉「仅 bootstrapCloneDone → 已完成」回落；无进度且无「已移入」日志则标「未开始」。
3. **私有仓不得回退 `task_tasks.owner_id`**。容器 token 带 `comment_id` → 读 `task_comments.created_by_id` → 用该用户发现 nested 并继承其 `task_git_identities`。他人的 `task_repo_identities` 不得顶替。

## 验证

```bash
cd taskCredentialService && go test ./application/ ./infrastructure/ -count=1 \
  -run 'CommentAuthor|FetchCommentAuthor|EnrichReposForCommentAuthor'
```

重建评论容器后，task-detail 日志应出现 `nested-git-repos fetched ... nested=34`，评论级进度为 `(n/35)` 而非 `(1/1)`。
