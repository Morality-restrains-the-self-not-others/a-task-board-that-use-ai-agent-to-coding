# v110 application-integration — 租户自建 GitLab OIDC SSO (current)

```mermaid
graph TD;
  taskFE["taskFE SPA"];
  gw["taskGateway / APISIX"];
  auth["taskAuth"];
  gitoauth["taskGitOauth"];
  tenantSvc["taskTenantService"];
  gitService["gitService 平台 GitLab"];
  selfGl["租户自建 GitLab"];
  oidcClient["auth_oidc_client"];
  tenantConn["git_oauth_tenant_gitlab_oauth_connections"];
  evEnabled["TenantGitLabOidcSsoEnabled"];
  plateauV109["Plateau v109"];
  plateauV110["Plateau v110 租户自建 GitLab OIDC SSO"];
  gapSso["Gap: 自建 GitLab 不能用平台账号 Web 登录"];
  wpSso["WP-tenant-selfhosted-gitlab-oidc-sso"];
  taskFE --> gw;
  gw --> gitoauth;
  gw --> auth;
  selfGl --> gw;
  gitService --> gw;
  auth --> tenantSvc;
  auth --> oidcClient;
  gitoauth --> tenantConn;
  auth --> evEnabled;
  plateauV109 --> gapSso;
  wpSso --|> gapSso;
  wpSso --|> plateauV110;
```
