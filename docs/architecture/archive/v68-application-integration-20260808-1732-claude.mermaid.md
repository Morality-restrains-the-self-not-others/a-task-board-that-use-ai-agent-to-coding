# v68 Application Integration — 浏览器插件 OIDC 白名单管理 (Target, 2026-08-08 17:32)

```mermaid
graph TD;
  gateway["APISIX Gateway (:18081) [MODIFIED v68: +/api/system-admin/oidc-extension/* 路由]"];
  taskFE["taskFE (SPA) [MODIFIED v68: SystemAdmin「浏览器插件」页 +/system-admin/oidc-extension/]"];
  taskAuth["taskAuth (:8003) [MODIFIED v68: +GET/PUT /api/system-admin/oidc-extension/ +seed managed_by='admin' 行 INSERT-only]"];
  taskAuthDB["taskAuth DB (+auth_oidc_client.managed_by 🆕 redirect_uris 写库)"];
  plateauV67["Plateau v67 — 任务帖存续期"];
  plateauV68["Plateau v68 — 插件 OIDC 白名单管理"];
  gapExt["Gap: 插件 ID 硬编码 conf + bootstrap 自愈覆盖管理员改库"];
  wpExt["WP-oidc-extension-admin"];
  taskFE --> gateway;
  gateway --> taskAuth;
  taskAuth ..> taskAuthDB;
  plateauV67 --> gapExt;
  wpExt --|> gapExt;
  wpExt --|> plateauV68;
```
