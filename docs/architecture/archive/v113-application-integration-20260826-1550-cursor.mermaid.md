# v113 application-integration — 动态 scene 码对账 (target)

```mermaid
graph TD;
  wxmp["微信服务号"];
  taskFE["taskFE SPA"];
  gw["taskGateway / APISIX"];
  auth["taskAuth"];
  ref["taskReferral"];
  ident["wechat_identity"];
  pend["auth_wechat_mp_subscribe_pending"];
  ticket["auth_wechat_mp_follow_ticket"];
  evSub["WECHAT_MP_SUBSCRIBED"];
  evConf["WECHAT_IDENTITY_CONFLICT"];
  p112["Plateau v112 静态码闸门"];
  p113["Plateau v113 动态 scene 票"];
  gapQr["Gap: 静态码无 scene，已关注无 SCAN"];
  wpQr["WP-referral-mp-dynamic-qr-ticket"];
  wxmp --> gw;
  gw --> auth;
  taskFE --> gw;
  gw --> ref;
  auth --> wxmp;
  auth --> ident;
  auth --> pend;
  auth --> ticket;
  auth --> evSub;
  auth --> evConf;
  ref --> auth;
  p112 --> gapQr;
  wpQr --|> gapQr;
  wpQr --|> p113;
```
