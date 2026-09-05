# v117 application-integration — 开放式邀请链接 (archived)

```mermaid
graph TD;
  taskFE["taskFE SPA"];
  gw["taskGateway / APISIX"];
  ten["taskTenantService"];
  ev["taskEvents"];
  inv["tenant_invitation"];
  red["tenant_invitation_redemption"];
  evCreated["INVITATION_CREATED"];
  evJoined["MEMBER_JOINED"];
  p116["Plateau v116 链接仅单次"];
  p117["Plateau v117 开放邀请"];
  gapOpen["Gap: 邀请链接只能使用一次"];
  wpOpen["WP-open-invite-link"];
  taskFE --> gw;
  gw --> ten;
  ten --> inv;
  ten --> red;
  ten --> evCreated;
  ten --> evJoined;
  ten --> ev;
  p116 --> gapOpen;
  wpOpen --|> gapOpen;
  wpOpen --|> p117;
```
