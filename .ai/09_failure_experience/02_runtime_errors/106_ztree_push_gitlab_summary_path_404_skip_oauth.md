# [运行时] ztree GitLab 推送：summary-for-user 404 被当成未绑定而跳过换票

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-21
- 编号：106
- 维护者：Trae AI 团队

## 现象

- 修完 [105](./105_ztree_push_git_identity_internal_gateway_sentinel.md) 后，同用户、同 Git 身份调 `POST /api/internal/layer-git-push/prepare` 不再 404，改为 **409**「未能换取 Git OAuth 凭据，无法推送到 HTTPS 远端」。
- 直接打 taskGitOauth `POST /api/internal/gitlab/oauth/access-for-user/`（`provider_key=gitlab:tencent-sh-1`）**200**，有 access_token。
- 同用户打 Cloud 用的 `POST /api/internal/gitlab/oauth/user-credential/summary-for-user/` → **404 page not found**。

## 根因

1. GitLab prepare 曾先调 `fetchGitOauthCredentialSummary`，`connected=false` 就 **continue 跳过** `access-for-user`。
2. Cloud 把 summary 打到旧路径 `/api/internal/{provider}/oauth/user-credential/summary-for-user/`。taskGitOauth **未注册**该路径（契约审计已标红）。404 被当成「未绑定」。
3. 正确路径是 `/api/internal/git-oauth/gitlab-credential-summary/`，该用户 `connected=true`。
4. GitHub 推送路径本来就直接换票，所以只有 **单仓 GitLab** 会踩这个门。

## 解决方案

1. `fetchGitOauthCredentialSummary` 改为 canonical ` /api/internal/git-oauth/{provider}-credential-summary/`。
2. GitLab 换票与 GitHub 一致：以 `access-for-user` 为 SSOT，summary 404 不再跳过换票。

## 验证

```bash
cd taskCloudService && go test ./src/ -count=1 \
  -run 'TestFetchGitOauthCredentialSummary_UsesCanonicalGitOauthPath|TestLayerGitPushPrepare_GitlabOauthWhenLegacySummary404'
curl -sS -X POST http://127.0.0.1:8018/api/internal/layer-git-push/prepare \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id":"<tid>","task_id":"<task>","layer_id":"L1","user_id":"<auth_user>","identity_id":"<gi_>","prefer_container_remote":false,"repo_url":"https://gitlab-tencent-sh-1.daydaymoney.com/..."}'
# 期望 HTTP 200，use_oauth_access_push=true，oauth_auth_by_repo 含 token
```

## 关联

- `.ai/09_failure_experience/02_runtime_errors/105_ztree_push_git_identity_internal_gateway_sentinel.md`
- `docs/superpowers/specs/2026-08-04-api-path-convention-audit.md`（summary-for-user → credential-summary）
- `taskCloudService/src/git_push_oauth.go`
- `taskCloudService/src/git_push_auth_context.go`
- `taskGitOauth/src/app.go`
