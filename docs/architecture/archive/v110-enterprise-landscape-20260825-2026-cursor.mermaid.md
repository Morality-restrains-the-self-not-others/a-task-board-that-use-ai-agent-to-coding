# v110 enterprise-landscape — 租户自建 GitLab OIDC SSO (current)

```mermaid
graph TD;
  admin["租户管理员"];
  member["租户成员"];
  bindGit["Git 网站授权绑定"];
  issueSso["签发平台 OIDC SSO"];
  loginGl["用平台账号登录自建 GitLab"];
  cfgOmni["在自建 GitLab 配置 OmniAuth"];
  fe["taskFE gitlab-connection"];
  auth["taskAuth"];
  gitoauth["taskGitOauth"];
  selfGl["租户自建 GitLab"];
  p109["Plateau v109"];
  p110["Plateau v110"];
  g["Gap: 自建 GitLab 仅有 OAuth App，无平台 Web SSO"];
  wp["WP-tenant-selfhosted-gitlab-oidc-sso"];
  admin --> issueSso;
  admin --> cfgOmni;
  admin --> fe;
  admin --> selfGl;
  member --> bindGit;
  member --> loginGl;
  member --> selfGl;
  fe --> auth;
  fe --> gitoauth;
  selfGl --> auth;
  p109 --> g;
  wp --|> g;
  wp --|> p110;
```
