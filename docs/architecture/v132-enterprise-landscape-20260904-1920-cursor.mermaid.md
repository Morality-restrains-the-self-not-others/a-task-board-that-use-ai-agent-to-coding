# v132 enterprise-landscape — WeChat MP egress on Host sh

```mermaid
flowchart TB
  User[推荐申请人]
  Auth[taskAuth on INFRA]
  Egress[wechat-mp-egress on Host sh]
  WX[WeChat MP API]
  User --> Auth --> Egress -->|1.117.67.121| WX
```
