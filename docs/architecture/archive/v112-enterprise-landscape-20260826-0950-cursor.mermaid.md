# v112 enterprise-landscape — 服务号关注闸门 (target)

```mermaid
graph TD;
  user["已登录用户"];
  wx["微信服务号"];
  fe["taskFE 推荐页"];
  auth["taskAuth"];
  ref["taskReferral"];
  ident["wechat_identity mp"];
  p111["Plateau v111"];
  p112["Plateau v112"];
  g["Gap: 申请不要求关注服务号"];
  wp["WP-referral-mp-follow-gate"];
  user --> wx;
  wx --> auth;
  user --> fe;
  fe --> auth;
  fe --> ref;
  auth --> ident;
  p111 --> g;
  wp --|> g;
  wp --|> p112;
```
