# v106 enterprise-landscape — 公网安全入口加固 (target)

```mermaid
graph TD;
  dev["开发者 / 外部用户"];
  secSvc["安全访问公网平台 🟡 MODIFIED"];
  edge["边缘 nginx (Host) 🟡 MODIFIED"];
  gw["taskGateway / APISIX"];
  taskFE["taskFE SPA"];
  lets["Let's Encrypt wildcard 🟡 MODIFIED"];
  plateauV104["Plateau v104"];
  plateauV106["Plateau v106 安全头+HTTPS 跳转"];
  gapSec["Gap: 公网响应无安全头、无强制 HTTPS"];
  wpSec["WP-security-headers-hsts"];
  dev --> secSvc;
  edge --> secSvc;
  edge --> gw;
  gw --> taskFE;
  lets --> edge;
  plateauV104 --> gapSec;
  wpSec --|> gapSec;
  wpSec --|> plateauV106;
```
