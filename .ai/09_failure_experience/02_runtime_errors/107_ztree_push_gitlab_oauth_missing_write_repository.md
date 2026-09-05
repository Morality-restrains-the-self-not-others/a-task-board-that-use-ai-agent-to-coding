# [运行时] ztree GitLab 推送：OAuth 缺 write_repository → HTTP Basic Access denied

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-21
- 编号：107
- 维护者：Trae AI 团队

## 现象

- 修完 [105](./105_ztree_push_git_identity_internal_gateway_sentinel.md) / [106](./106_ztree_push_gitlab_summary_path_404_skip_oauth.md) 后，公网「提交并创建PR」越过身份 404 与换票 409。
- 容器 `oauth-access-push` 真实 `git push`，GitLab 返回：`remote: HTTP Basic: Access denied`。
- `data-traceId`：`bf70fdd7-bb60-4721-9201-7b51ce4c4897`（当时 Loki 无数据；Cloud 本地日志已轮转）。
- Gateway `auth_internal_bypass.user_id` 已是真实用户，不是哨兵 `internal_gateway`。

## 根因

1. 换到的 GitLab OAuth token 的 scope 为 `read_repository api read_user`，**没有 `write_repository`**。
2. GitLab 文档：`api` 管 REST；`read_repository` 只读 git-over-HTTP；**git push 必须 `write_repository`**。
3. 区域 YAML（`gitlab:tencent-sh-1` / `gitlab:daydaymoney-gitlab`）把 scope 配成了薄集；localhost/synology 已含 `write_repository`。
4. `/oauth/token/info` 可证：`scope=["read_repository","api","read_user"]`；`GET /api/v4/user` 仍 200。

## 解决方案

1. 两棵 provider 树的 GitLab YAML `target.scope` 改为 `read_repository write_repository api read_user`。
2. `ResolveGitLabAuthorizeContext` 空 scope 回退与租户默认 `domain.DefaultTenantGitLabScope` 对齐。
3. `bash gitService/scripts/sync_local_oauth_app_scopes.sh` 把 Doorkeeper Application 范围同步到各区域容器。
4. **存量用户必须重新授权**（refresh 不会自动加上新 scope），再点「提交并创建PR」。

## 验证

```bash
# token/info 应含 write_repository
# 公网同一任务再点「提交并创建PR」：GitLab 出现 feature 分支 + MR，不再 HTTP Basic Access denied
```

## 关联

- `.ai/09_failure_experience/02_runtime_errors/105_ztree_push_git_identity_internal_gateway_sentinel.md`
- `.ai/09_failure_experience/02_runtime_errors/106_ztree_push_gitlab_summary_path_404_skip_oauth.md`
- `conf/auth/git-oauth/providers/http-gitlab-tencent-sh-1.yaml`
- `taskGitOauth/infrastructure/oauth_resolve.go`
- `gitService/scripts/sync_local_oauth_app_scopes.sh`
- 同症状后续根因：[108](./108_ztree_push_gitlab_borrow_other_ce_token.md)（借其它 CE token）
