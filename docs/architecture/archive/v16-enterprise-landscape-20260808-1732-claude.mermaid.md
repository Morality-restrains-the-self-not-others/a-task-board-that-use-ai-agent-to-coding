# v16 Enterprise Landscape — 插件 OIDC 白名单管理 (Target, 2026-08-08 17:32)

```mermaid
graph TD;
  feSvc["taskFE (Vue SPA) [MODIFIED v16: SystemAdmin「浏览器插件」管理页]"];
  gateway["API Gateway (APISIX :18081) [MODIFIED v16: +/api/system-admin/oidc-extension/* 路由]"];
  authSvc["taskAuth (Go :8003) [MODIFIED v16: +GET/PUT /api/system-admin/oidc-extension/ +seed INSERT-only]"];
  authDB["taskAuth DB (+auth_oidc_client.managed_by 🆕)"];
  plateauV15["Plateau v15 — 任务帖存续期"];
  plateauV16["Plateau v16 — 插件 OIDC 白名单管理"];
  gapExt["Gap: 插件 ID 硬编码 conf + bootstrap 自愈覆盖管理员改库"];
  wpExt["WP-oidc-extension-admin"];
  feSvc --> gateway;
  gateway --> authSvc;
  authSvc ..> authDB;
  plateauV15 --> gapExt;
  wpExt --|> gapExt;
  wpExt --|> plateauV16;
```
