# v117 enterprise-landscape — 开放式邀请链接 (archived)

```mermaid
graph TD;
  inviter["邀请人"];
  joiners["多名被邀请人"];
  createOpen["创建开放邀请链接"];
  multiJoin["多人点击同一链接加入"];
  fe["taskFE"];
  ten["taskTenantService"];
  p116["Plateau v116"];
  p117["Plateau v117"];
  g["Gap: 链接仅单次"];
  wp["WP-open-invite-link"];
  inviter --> createOpen;
  joiners --> multiJoin;
  inviter --> fe;
  joiners --> fe;
  fe --> ten;
  p116 --> g;
  wp --|> g;
  wp --|> p117;
```
