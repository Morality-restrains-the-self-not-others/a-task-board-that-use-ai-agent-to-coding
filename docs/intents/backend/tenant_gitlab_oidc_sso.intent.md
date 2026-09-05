# 功能意图：租户自建 GitLab 平台 OIDC SSO 签发

- **日期**: 2026-08-25
- **状态**: 实现中

## 背景与目标

租户管理员在 gitlab-connection 为自建 GitLab 签发 taskAuth OIDC client，使成员可用平台账号登录客户 GitLab Web。

## 范围

- `GET/PUT/DELETE /api/tenant/{tid}/gitlab-oidc-sso/`
- `POST /api/tenant/{tid}/gitlab-oidc-sso/rotate/`
- Authorize 成员闸门（`gitlab-tenant-*`）
- 协议端点 `{issuer}/api/oidc/{tid}/authorize|token|userinfo|jwks`（ADR-0044）

## 约束

- client_id = `gitlab-tenant-{tid}`；禁止 `gitlab-git-service` 前缀
- 写：租户管理员 + `settings.gitlab.main` operate
- secret 仅签发/轮换响应一次
- 数据 owner：taskAuth `auth_oidc_client`

## 验收标准

1. 非成员 authorize → access_denied
2. GET 永不返回 secret
3. PUT 无 Path A 或 base_url 不一致 → 400
4. 公网 HTTP callback 允许签发（redirect_uri 与 Path A `base_url` 同 scheme）；PUT 成功时 slog warn `tenant_gitlab_oidc_sso_http_redirect`。非 http/https 或 callback path 非法 → 400

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 键 |
|----------|--------|-----|
| 管理员签发 SSO | `TenantGitLabOidcSsoEnabled` | company_id |
| 管理员轮换密钥 | `TenantGitLabOidcSsoSecretRotated` | company_id |
| 管理员吊销 SSO | `TenantGitLabOidcSsoDisabled` | company_id |
| 成员完成 GitLab SSO 登录 | — | 与平台 gitService OIDC 一致，不发新事件 |

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-25 | 初稿（goal 流水线） |
| 2026-08-25 | ADR-0044：片段与路由改为 `/api/oidc/{tid}/*`，`discovery: false` |
| 2026-08-26 | HTTP GitLab 签发 400 改为中文 HTTPS 指引；结构化 warn `tenant_gitlab_oidc_sso_insecure_redirect` |
| 2026-08-26 | 产品确认：公网 HTTP GitLab 允许签发；warn 改为 `tenant_gitlab_oidc_sso_http_redirect` |
