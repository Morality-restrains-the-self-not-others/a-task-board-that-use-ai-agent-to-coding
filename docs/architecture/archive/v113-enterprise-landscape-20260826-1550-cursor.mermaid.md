# v113 enterprise-landscape — 动态 scene 码 (target)

```mermaid
graph TD;
  user["已登录用户"];
  wx["微信服务号"];
  fe["taskFE 推荐页"];
  auth["taskAuth"];
  ref["taskReferral"];
  ident["wechat_identity mp"];
  ticket["auth_wechat_mp_follow_ticket"];
  p112["Plateau v112"];
  p113["Plateau v113"];
  g["Gap: 静态码无法对账已关注用户"];
  wp["WP-referral-mp-dynamic-qr-ticket"];
  user --> fe;
  fe --> auth;
  auth --> wx;
  user --> wx;
  wx --> auth;
  fe --> ref;
  auth --> ident;
  auth --> ticket;
  p112 --> g;
  wp --|> g;
  wp --|> p113;
```
