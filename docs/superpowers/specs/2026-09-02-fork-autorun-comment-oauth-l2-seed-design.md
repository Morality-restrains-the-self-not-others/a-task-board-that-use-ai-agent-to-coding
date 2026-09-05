# 设计：Fork 自动运行评论从源任务种下 Git OAuth L2

- **日期**: 2026-09-02
- **状态**: 已批准（/goal 零交互采用）
- **范围**: taskTaskService 评论 L2 种下；不改换票协议、不新增 HTTP 端点

## 问题

公网任务 `task_882926898668138496`、评论 `cmt_882926907161604096`（【自动运行】，Fork 派生）层图「push 失败」：

`layer-github-oauth-access-tokens` → HTTP 409 `BINDING_MISSING`，`缺少绑定: gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work`，`github_auth_by_repo: {}`。

无 `data-traceId`。Loki `{job=~".+"} |= "cmt_882926907161604096"` 48h 空（日志已滚或未带 comment_id）。

## 根因

ADR-0049：换票只认**本条评论**的 `oauth_gitsite`。Fork 自动运行评论的身份来自前端 `normalizeCreateTaskRepoIdentities`，**只提交** `{repo_url, git_identity_id}`。

种下优先级（ADR-0055）为：

1. 本会话 `grant_ticket`（一次性；源任务创建时往往已消费）
2. `project_git_oauth_grant`（仅项目详情授权过才有）

Fork 弹窗「已绑定」可来自**仍留在 session 的已消费 ticket**。后端 consume 失败、项目 L2 也没有时，新【自动运行】评论无 `oauth_gitsite` → 换票 409。源任务评论上用户已经授过权。

## 方案（采用）

在 `prepareAutoRunRepoIdentities` 增加第三源：**源任务（`fork_from_id`）上、同一 `created_by_id` 的评论 L2**。

优先级：ticket → 项目 L2 → **Fork 源评论 L2**。

约束：

- 仅同用户；禁止同事/他人 L2
- 按 gitsite 对齐（含源评论站点级空 `repo_url` 行）
- 已有 `oauth_gitsite` 不覆盖
- 发布 `COMMENT_GIT_OAUTH_GRANTED`，`via=fork_source_comment_l2_seed`，幂等键 `grant:comment:{commentID}:{userID}:{gitsite}`

存量修复：`container-snapshot` 读评论身份时，若缺 L2 则同样种下并 **UPDATE**（幂等）。下次换票即可带上 L2，无需用户再点 OAuth。芯片上的旧 `last_push_error` 在下一次成功 push 后由容器清除。

## 非方案

- 换票绕过 ADR-0049（禁止）
- 把已消费 session ticket 当成仍有效（UI 误导，不修根因）
- 从源评论拷贝他人的 grant

## 架构

当前基线 v128（阿里云 GitLab pending_node），与本增量无关。

**不更新** `docs/architecture/` ArchiMate：无新服务、无新事件类型、无新表；仅扩展既有 `COMMENT_GIT_OAUTH_GRANTED` 的 `via`。

## Code Review Graph

CRG unavailable: 仓库无 `.codegraph/` 索引。

## 角色权限

| 改动点 | 主体 | 资源 | 操作 | 检查 | 结论 |
|--------|------|------|------|------|------|
| `applyForkSourceCommentL2SeedToIdentities` | Fork 操作者（评论 `created_by_id`） | 该【自动运行】评论 L2 | write | 仅同源任务、同 user_id 的 L2 | ✅ 充分 |
| snapshot GET 补种 | credential 内部调用 | 同上 | write（缺 L2 时） | internal secret + 同用户 | ✅ 充分 |

无新公开 API。不引入新角色。

## NFR / 幂等

| 路径 | 重复边界 | 键 | 级别 |
|------|----------|-----|------|
| Kafka `COMMENT_GIT_OAUTH_GRANTED` | 该 comment × user × gitsite | `grant:comment:{id}:{user}:{gitsite}` | L1 |
| snapshot 补种 UPDATE | 已有 oauth_gitsite 则跳过 | 评论 JSON 字段 | L1 |

无资金路径。路径已含 task/comment id。

## 验收

1. Fork 任务 `fork_from` 源评论（同 user）有 GitLab L2、新评论仅有 git_identity_id 时，ensure 后 JSON 含 `oauth_gitsite`。
2. 不同 user 不种下。
3. 无 `fork_from` 不种下。
4. 已有 L2 不被覆盖。
5. snapshot 对缺 L2 的 Fork 评论补种并落库。
6. 事件 `via=fork_source_comment_l2_seed`。
