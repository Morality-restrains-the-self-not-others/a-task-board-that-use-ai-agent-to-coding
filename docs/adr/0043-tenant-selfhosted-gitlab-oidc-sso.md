# ADR-0043: 租户自建 GitLab 使用独立 OIDC client 与成员闸门做平台 SSO

- **Status:** accepted
- **Date:** 2026-08-25
- **Author:** cursor
- **Deciders:** 工程团队（头脑风暴总体设计已批准）

---

## Context

租户可在 `/tenant/{tid}/settings/gitlab-connection/` 登记**自建 GitLab OAuth Application**（链路 A，v30 / `taskGitOauth`），供成员「Git 网站授权」绑定，平台据此 clone/push。这与**平台 gitService** 的登录模型不同：平台 GitLab 是 OmniAuth RP，taskAuth 是 OIDC IdP（ADR-0016，client_id `gitlab-git-service*`）。

用户需要自建 GitLab Web 也能用平台账号登录。若复用 `gitlab-git-service*` 或给同一 client 加多条 redirect_uri：

1. `oidc_region_gate` 按 `gitlab-git-service` 前缀套用区域 `access_mode` / tester 闸门，会误伤客户实例；
2. 任意平台用户完成 SSO 即可在客户 GitLab 上被 `omniauth_block_auto_created_users=false` 自动建号；
3. 平台无法写入客户 `gitlab.rb`，只能签发凭据并提供片段。

ADR-0016 **只约束平台运营的 gitService**，不强制客户关闭注册/账密。

## Decision

We will treat **tenant self-hosted GitLab as a distinct OIDC relying party**:

1. **独立 client_id** `gitlab-tenant-{company_id}`，`auth_oidc_client.managed_by=tenant`，带 `owner_company_id` / `purpose=tenant_gitlab_sso`。**禁止**使用 `gitlab-git-service` 前缀。
2. **Authorize 成员闸门（fail-closed）**：仅该租户在籍成员可获得授权码；解析走 taskTenantService `members/resolve`（与 taskGitOauth 同源）。
3. **平台不代配客户 GitLab**：设置页返回 issuer、secret（一次性）、OmniAuth 片段；管理员在自建实例 `gitlab.rb` 自行 `reconfigure`。
4. **链路 A 与链路 B 独立**：OAuth Application（GitLab 为 AS）与 OIDC SSO（taskAuth 为 IdP）互不替代；不把 Application 的 `openid` scope 当成 Web SSO。
5. **redirect_uri** 固定为 `{base_url}/users/auth/openid_connect/callback`，生产环境要求 HTTPS。

## Alternatives Considered

### Alternative 1: 给 `gitlab-git-service` 追加客户 redirect_uri

- **Pros:** 零新 client
- **Cons:** 共享 secret；区域闸门无法区分；任意 RP 共用一个受众
- **Why rejected:** 安全面与 ADR-0041 区域模式耦合

### Alternative 2: `gitlab-git-service-tenant-{id}`

- **Pros:** 命名接近平台实例
- **Cons:** 命中 `oidcGitlabClientPrefix`，被当成平台区域 GitLab
- **Why rejected:** 前缀冲突

### Alternative 3: 仅文档，超管手工 bootstrapClients

- **Pros:** 无新 API
- **Cons:** conf 膨胀、secret 进仓库、无法自助
- **Why rejected:** 不满足租户自助

## Consequences

### Positive

- 客户 GitLab SSO 与平台区域 GitLab SSO 闸门隔离
- 非本租户平台用户无法拿到 code
- 不把 ADR-0016 强加给客户实例

### Negative / Trade-offs

- 客户必须能改 OmniAuth 且 GitLab **出站**可达公网 issuer
- 客户仍可保留账密，身份可能双轨
- taskAuth 新增租户级 OIDC CRUD 与事件

### Mitigations

- 设置页写明网络与 HTTPS 前提；HTTP 自建默认拒签
- bootstrap seed 跳过 `managed_by=tenant` 行
- 轮换 secret 立即失效旧凭据

## References

- 设计：`docs/superpowers/specs/2026-08-25-tenant-selfhosted-gitlab-oidc-sso-design.md`
- [ADR-0014](0014-pluggable-multi-region-gitlab.md)、[ADR-0016](0016-gitlab-sso-only-no-self-signup.md)、[ADR-0041](0041-tester-role-gitlab-region-access-mode.md)
- `.ai/01_project_constraints/55_gitlab_sso_only_no_self_signup.md`（仅平台 gitService）
