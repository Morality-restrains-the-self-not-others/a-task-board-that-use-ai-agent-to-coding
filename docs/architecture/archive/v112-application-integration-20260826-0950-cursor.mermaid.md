# v112 application-integration — 服务号关注闸门 (target)

```mermaid
graph TD;
  wxmp["微信服务号"];
  taskFE["taskFE SPA"];
  gw["taskGateway / APISIX"];
  auth["taskAuth"];
  ref["taskReferral"];
  ident["wechat_identity"];
  pend["auth_wechat_mp_subscribe_pending"];
  evSub["WECHAT_MP_SUBSCRIBED"];
  plateauV111["Plateau v111"];
  plateauV112["Plateau v112 服务号关注闸门"];
  gapMp["Gap: 无服务号回调，分账缺 mp openid"];
  wpMp["WP-referral-mp-follow-gate"];
  wxmp --> gw;
  gw --> auth;
  taskFE --> gw;
  gw --> ref;
  auth --> ident;
  auth --> pend;
  auth --> evSub;
  ref --> auth;
  plateauV111 --> gapMp;
  wpMp --|> gapMp;
  wpMp --|> plateauV112;
```
