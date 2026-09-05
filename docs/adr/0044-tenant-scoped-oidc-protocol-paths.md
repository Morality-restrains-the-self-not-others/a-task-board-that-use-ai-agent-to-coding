# ADR-0044: 租户自建 GitLab OIDC 协议端点路径携带 tenantId

- **Status:** accepted
- **Date:** 2026-08-25
- **Author:** cursor
- **Deciders:** 工程团队（goal 零交互，沿用 ADR-0043 租户 SSO）

---

## Context

ADR-0043 为租户自建 GitLab 签发独立 OIDC client，设置页 OmniAuth 片段把 `authorization_endpoint` / `token_endpoint` / `userinfo_endpoint` / `jwks_uri` 写成**全局** `{issuer}/api/oidc/{authorize|token|userinfo|jwks}`。

这与平台 gitService、Chrome 插件、厂商门户共用同一组无租户键路径。NFR 路径分片审视要求租户 SSO 协议面可按 `company_id` 扩展；全局 discovery（`discovery: true`）还会用 `/.well-known/openid-configuration` **覆盖**片段里的显式 URL，租户键永远不会被 GitLab 使用。

## Decision

We will:

1. **租户 SSO 片段**使用 `{issuer}/api/oidc/{tenantId}/authorize|token|userinfo|jwks`，并设 `discovery: false`，使 OmniAuth 采用显式端点。
2. **taskAuth 注册同路径**并校验 path 上的 `tenantId` 与 client 的 `owner_company_id` 一致；不一致返回 `unauthorized_client`（authorize/token）或 `invalid_token`（userinfo）。
3. **JWKS** 租户路径仍返回当前全局密钥集（为将来按租户拆钥预留 URL，本决策不改签名材料）。
4. **保留**全局 `/api/oidc/*` 供平台 RP（`gitlab-git-service*`、chrome-extension、ai-provider）。issuer 与 ID token `iss` **保持全局**，避免改 claim。
5. 登录回跳 `next=` 允许相对/绝对 `/api/oidc/{tenantId}/authorize`。

## Alternatives Considered

### Alternative 1: 仅改 snippet 文案、后端仍只听全局路径

- **Pros:** 改动面小
- **Cons:** 客户按片段配置后 404；无法按租户扩展
- **Why rejected:** 文案与路由必须一致

### Alternative 2: 租户级 issuer `{issuer}/api/oidc/{tid}` + discovery

- **Pros:** 符合「issuer 含 path 则 well-known 挂在 issuer 下」
- **Cons:** ID token `iss` 必须随租户变；所有 RP 校验与现网全局 issuer 分叉
- **Why rejected:** 本增量只要协议 URL 可分片，不改 iss 契约

### Alternative 3: 把全部 OIDC（含平台 GitLab）迁到带 tenant 的路径

- **Pros:** 路径完全统一
- **Cons:** 破坏 gitService / Chrome 插件 / 厂商门户
- **Why rejected:** 平台 RP 不是租户资源

## Consequences

### Positive

- 设置页片段与真实路由一致；GitLab 出站打到带 `tenantId` 的 URL
- 错租户 path + 他租户 client 被拒绝，降低误配面
- 全局 `/api/oidc/*` 仍可用，已发出的平台 OmniAuth 不受影响

### Negative / Trade-offs

- 已复制旧片段（全局 URL + `discovery: true`）的客户需重新粘贴
- 网关多 4 条通配路由；JWKS 尚未真正按租户分钥

### Mitigations

- GET SSO 始终返回最新片段；轮换密钥时一并展示
- 全局路径保留作兼容与平台 RP

## References

- [ADR-0043](0043-tenant-selfhosted-gitlab-oidc-sso.md)
- 设计：`docs/superpowers/specs/2026-08-25-tenant-selfhosted-gitlab-oidc-sso-design.md`
- `.ai/01_project_constraints/48_nfr_path_shard_id_scalability.md`
