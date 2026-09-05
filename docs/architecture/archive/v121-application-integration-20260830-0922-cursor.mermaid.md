# v121 application-integration — 意见与建议链接 (current)

```mermaid
graph TD;
  fe["taskFE SPA"];
  gw["taskGateway / APISIX"];
  bill["taskBill"];
  ev["taskEvents"];
  grp["billing_feedback_link_group"];
  thr["billing_feedback_link_threshold"];
  lnk["billing_feedback_link"];
  kind["billing_feedback_resource_kind"];
  usage["billing grant/usage/account"];
  evC["FEEDBACK_LINK_GROUP_CREATED"];
  evU["FEEDBACK_LINK_GROUP_UPDATED"];
  evD["FEEDBACK_LINK_GROUP_DELETED"];
  p120["Plateau v120 任务项目修订"];
  p121["Plateau v121 消耗门槛链接"];
  gap["Gap: 侧栏无法按租户消耗分层展示外链"];
  wp["WP-tenant-feedback-links"];
  fe --> gw;
  gw --> bill;
  bill --> grp;
  bill --> thr;
  bill --> lnk;
  bill --> kind;
  bill --> usage;
  bill --> evC;
  bill --> evU;
  bill --> evD;
  bill --> ev;
  p120 --> gap;
  wp --|> gap;
  wp --|> p121;
```
