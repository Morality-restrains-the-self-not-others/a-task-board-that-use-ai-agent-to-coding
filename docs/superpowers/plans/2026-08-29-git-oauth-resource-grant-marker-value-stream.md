# 价值流：Git OAuth 资源使用标记

- **日期**: 2026-08-29
- **设计**: `docs/superpowers/specs/2026-08-29-git-oauth-resource-grant-marker-design.md`
- **配置**: `conf/value-stream.yaml` stream `git-oauth-resource-grant-marker`

## Related Value Streams

| 现流 | 关系 |
|------|------|
| `gitoauth-binding-state-persistence` | L1 不变；本流在回调后**额外**写 L2 |
| `project-detail-oauth-token-status-aware` | 徽章从「L1 connected」改为 L2+probe |
| `project-detail-nested-git-repos-oauth` | 子仓行同样读 L2 |
| `create_task_oauth_bind`（意图，YAML 未单列） | 门禁改为启用者会话 ticket，不看项目 L2 |

**修改而非绿场**：同一用户 OAuth 主路径，增量是使用闸门。

## 用户可感知价值（顺序）

1. **项目页不会再因「别处绑过」显示已授权** — 无本项目 L2 则需要授权
2. **OAuth 回流后项目页变成已授权**（可写探测通过）
3. **勾选自动运行必须由创建者完成 OAuth**；ticket 打在【自动运行】评论，不抄项目
4. **排队开火用评论作者票**，不随 Owner 改派

## 增量切片

| # | 增量 | 交付价值 | 依赖 |
|---|------|----------|------|
| I1 | 项目 L2 表 + token_status 无标记 | 徽章正确 | — |
| I2 | 回调 + start query 打项目标记 | 用户能授权项目 | I1 |
| I3 | 评论 JSON grant + grant_ticket | 自动运行/发评可打标 | I1 |
| I4 | 换票方先查 L2 | clone/push 安全 | I2/I3 |
| I5 | 排队 UserID=评论作者 | 延迟开火正确 | I3 |
| I6 | FE 门禁与徽章文案 | 可操作完成 | I2–I4 |
| I7 | 领域事件 | 审计/意图门禁 | I2/I3 |

## YAML fields（三元组）

- `taskProjectService.project_git_oauth_grant.gitsite`
- `taskProjectService.project_git_oauth_grant.remote_user_id`
- `taskProjectService.project_git_oauth_grant.task2app_user_id`
- `taskTaskService.task_comments.oauth_granted_at`（JSON `repo_identities[].oauth_granted_at`）
- `taskGitOauth.git_oauth_grant_ticket.id`

## 测试影响

- 扩展 `git_repo_validate` / FE `gitRepoOAuthStatusUtils` / `createTaskOauthGate`
- 新增 grant upsert、ticket consume、queued UserID 单测
- 更新 `create_task_oauth_bind` 意图文案
