# DDD：租户自建 GitLab OIDC SSO

- **日期**: 2026-08-25
- **限界上下文**: Identity / OIDC（taskAuth）
- **NFR**: `docs/superpowers/plans/2026-08-25-tenant-selfhosted-gitlab-oidc-sso-nfr-clarification.md`

## 聚合

**TenantGitLabOidcClient**（一对一租户）

- 标识：`ClientID` = `gitlab-tenant-{company_id}`
- 属性：`OwnerCompanyID`、`ManagedBy=tenant`、`Purpose=tenant_gitlab_sso`、`RedirectURIs`、`SecretHash`
- 不变量：
  1. ClientID 前缀不得为 `gitlab-git-service`
  2. 生产 RedirectURI 必须 HTTPS 且路径 `/users/auth/openid_connect/callback`
  3. 明文 secret 不入库、不出现在 GET

## 领域服务

- `DecideAuthorizeMembership(userID, ownerCompanyID, isMember, lookupErr)` → allow | deny(access_denied)
  - lookupErr 或 !isMember → deny（含超管非成员）
- `RedirectURIFromBaseURL(baseURL)` / `ValidateProductionRedirectURI`
- `OmniAuthSnippet(issuer, clientID, redirectURI)`

## 端口

- `OidcClientStore`：Load / InsertTenant / UpdateSecret / DeleteByClientID
- `TenantMembershipPort`：Resolve(userID, companyID) → (memberID, isAdmin, err)
- `PathAConnectionPort`：LoadBaseURL(companyID) → (configured, baseURL, err)
- `DomainEventPublisher`：Enabled / SecretRotated / Disabled

## 领域事件

| 事件 | 键 | 数据（无 secret） |
|------|-----|-------------------|
| `TenantGitLabOidcSsoEnabled` | company_id | client_id, redirect_uri, actor_user_id |
| `TenantGitLabOidcSsoSecretRotated` | company_id | client_id, actor_user_id |
| `TenantGitLabOidcSsoDisabled` | company_id | client_id, actor_user_id |

SSO 登录本身不发新事件（与平台 gitService OIDC 一致）。

## 适配器落点

- MySQL `auth_oidc_client`（taskAuth owner）
- HTTP → taskTenantService `GET /api/internal/tenant/members/resolve`
- HTTP → taskGitOauth `GET /api/internal/git-oauth/gitlab-tenant-connection/`
- Kafka `publishDomainEventKafka`

领域层文件：`taskAuth/domain/tenant_gitlab_oidc_sso.go`（无 DB/HTTP import）。
