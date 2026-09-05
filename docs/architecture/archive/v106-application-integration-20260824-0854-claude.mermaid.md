# v106 application-integration — 安全响应头 + HTTP→HTTPS 强制跳转 (target)

```mermaid
graph TD;
  edge["边缘 nginx (Host :443) 🟡 MODIFIED"];
  gw["taskGateway / APISIX"];
  taskFE["taskFE SPA"];
  cloud["taskCloudService"];
  django["saas-backend (Django legacy)"];
  cHSTS["HSTS 阶梯灰度 🟢 NEW"];
  cCSP["CSP Report-Only 灰度 🟢 NEW"];
  cRef["Referrer-Policy 保留 Origin 🟢 NEW"];
  plateauV104["Plateau v104"];
  plateauV106["Plateau v106 安全头+HTTPS 跳转"];
  gapSec["Gap: 公网响应无安全头、无强制 HTTPS"];
  wpSec["WP-security-headers-hsts"];
  edge --> gw;
  gw --> taskFE;
  gw --> cloud;
  gw --> django;
  cHSTS ..> edge;
  cCSP ..> edge;
  cRef ..> edge;
  plateauV104 --> gapSec;
  wpSec --|> gapSec;
  wpSec --|> plateauV106;
```
