# 测试意图：gitlab-connection SSO UI

- **日期**: 2026-08-25
- **对应功能意图**: `docs/intents/frontend/tenant_gitlab_oidc_sso.intent.md`

## 用例

1. 加载 GET `configured:false` 显示启用按钮
2. 启用 PUT 带 Idempotency-Key；展示一次性 secret
3. 再 GET 不展示 secret
4. 链路 A 帮助含 Redirect URI / scopes、不含把 openid 当 Path A
5. 无 region operate 时写按钮不可用
6. 登录回跳 `next` 接受 `/api/oidc/{tid}/authorize`，以及 apex/`www`/`api` 同站绝对 URL；拒绝 `/api/oidc/../authorize`
7. `pathABaseUrl` 为 `http://115.29.110.74` 时出现 `gitlab-oidc-sso-http-warning`，签发按钮可用，点击发 PUT
8. `gitlab-oidc-sso-where` 含 `/etc/gitlab/gitlab.rb`、`identifier`、`gitlab-ctl reconfigure`；签发后 `gitlab-oidc-sso-secret-hint` + 片段含 `secret:`

## 文件

- `taskFE/app/src/views/WorkspaceSettingsGitlabOidcSso.test.js`
- `taskFE/app/src/utils/gitlabOidcSsoSnippet.test.js`
- `taskFE/app/src/views/WorkspaceSettingsGitlabConnection.test.js`
- `taskFE/app/src/utils/oidcResumeUrl.test.js`
- `taskFE/app/src/utils/gitlabOidcSsoHttps.test.js`

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-25 | 初稿 |
| 2026-08-25 | ADR-0044 租户路径 resume 测例 |
| 2026-08-26 | 公网 HTTP 改为可签发 + http-warning |
| 2026-08-26 | 小白放置说明：gitlab.rb 字段对应与一次性 secret 写入片段 |
| 2026-08-26 | GitLab SSO 登录 next：www/api 与 apex 视为同站，登录后能回到 authorize |
