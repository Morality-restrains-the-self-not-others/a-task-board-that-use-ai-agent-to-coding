# v55 Application Integration — Gateway-Controlled Frontend Routing

```mermaid
graph TD;
  browser["Browser (www.daydaymoney.com)"];
  nginxNode["Nginx (:443)"];
  gateway["APISIX Gateway"];
  apiRoutes["/api/* Routes"];
  spaCatchAll["SPA catch-all (priority 10)"];
  oidcDiscovery[".well-known/openid-configuration"];
  taskAuth["taskAuth (:8003)"];
  taskProjectSvc["taskProjectService (:8016)"];
  taskCloudSvc["taskCloudService (:8018)"];
  taskBillSvc["taskBill (:8004)"];
  django["Django (:8001)"];
  taskFE["taskFE SPA Server (:3000)"];
  spaAssets["taskFE/static/"];
  vueRouter["Vue Router (Client-Side)"];
  loginPage["Login.vue"];
  projectsPage["Projects.vue"];
  oldSpaRoute["auth-login-page route [DEPRECATED]"];
  oldDefault["django-default catch-all [DEPRECATED]"];
  P54["Plateau v54 (current)"];
  P55["Plateau v55 (target)"];
  GapRouting["Gap: TaskGateway 不掌握非 API 路由"];
  WP55["WP: v55 Gateway-Controlled Frontend Routing"];
  browser --> nginxNode;
  nginxNode --> gateway;
  gateway --> apiRoutes;
  apiRoutes --> taskAuth;
  apiRoutes --> taskProjectSvc;
  apiRoutes --> taskCloudSvc;
  apiRoutes --> taskBillSvc;
  apiRoutes --> django;
  gateway --> oidcDiscovery;
  gateway --> spaCatchAll;
  spaCatchAll --> taskFE;
  taskFE ..> spaAssets;
  taskFE --> browser;
  taskFE --> vueRouter;
  vueRouter --> loginPage;
  vueRouter --> projectsPage;
  vueRouter --> gateway;
  P54 --> GapRouting;
  WP55 --|> GapRouting;
  WP55 --|> P55;
  oldSpaRoute --- gateway;
  oldDefault --- gateway;
```

## 图例

| 符号 | 含义 |
|------|------|
| 🟢 绿色节点 | [NEW] v55 新增组件 |
| 🟡 黄色节点 | [MODIFIED] v55 修改组件 |
| 🔴 红色节点 | [DEPRECATED] v55 废弃组件 |
| `-->` 实线箭头 | Rel_Flow（请求流） |
| `..>` 虚线箭头 | Rel_Access（数据访问） |
| `--|>` 空心箭头 | Rel_Realization（实现/关闭 Gap） |

## 请求流对比

### Before (v54) — GET /auth/login
```
Browser → Nginx location / → Django :8001 → _serve_spa_shell() → index.html
```

### After (v55) — GET /auth/login
```
Browser → Nginx location / → APISIX :18081 → spa-catch-all → taskFE :3000 → index.html → Vue Router → Login.vue
```

### API (unchanged) — POST /api/auth/
```
Browser → Nginx location / → APISIX :18081 → /api/auth/ route → taskAuth :8003 → JSON token
```
