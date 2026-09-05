# 角色权限分析：租户级自建 GitLab OAuth 连接

**日期：** 2026-07-15  
**基于：** `2026-07-15-tenant-gitlab-oauth-connection-design.md`

## 角色

| 角色 | 说明 |
|------|------|
| tenant_admin | 公司管理员；可配置/删除自建 GitLab OAuth App |
| tenant_member | 租户成员；可读连接摘要（无 secret）；可对本租户 provider 发起个人 OAuth |
| 登录用户（本人） | git-site-oauth 绑定仅本人 |
| 内部服务 | Django catalog → taskGitOauth internal resolve（bridge secret） |

## 端点权限矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 检查 | 建议 |
|--------|------|----------|------|------|------|
| GET `/api/tenant/{tid}/gitlab-oauth-connection/` | tenant_member | Tenant | read | 成员 + company_id=tid | secret 永不回传 |
| PUT 同上 | tenant_admin | Tenant | write | **ensureTenantAdmin** + tid | 校验 base_url https/http |
| DELETE 同上 | tenant_admin | Tenant | delete | ensureTenantAdmin | 级联解绑仅该 provider_key |
| GET providers（扩展） | 登录用户 | Tenant/User | read | IsAuthenticated | 仅合并**当前** company 连接 |
| gitlab oauth start（tenant-*） | 本人 | User | write | JWT / X-User-Id | SP 必须对应用户当前租户已配置 |
| internal resolve | 内部 | Tenant | read | bridge secret | 禁公网 |

## IDOR / 跨租户

- 路径 `tid` 必须与连接 `company_id` 一致；禁止用 body 覆盖 company_id。
- `service_provider=tenant-{other}`：start 时校验连接存在且 active；catalog 不泄露他租户。
- DELETE 级联仅删 `provider=gitlab:tenant-{tid}` 行。

## 审计

- PUT/DELETE 结构化日志（禁打印 client_secret）。
- Kafka：Upserted / Deleted（含 company_id，无 secret）。
