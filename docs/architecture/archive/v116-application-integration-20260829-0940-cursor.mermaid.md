# v116 application-integration — 邮件邀请退订 (target)

```mermaid
graph TD;
  taskFE["taskFE SPA"];
  gw["taskGateway / APISIX"];
  auth["taskAuth"];
  ten["taskTenantService"];
  ev["taskEvents"];
  unsub["auth_email_unsubscription"];
  evUnsub["EMAIL_UNSUBSCRIBED"];
  p115["Plateau v115 邀请只发信"];
  p116["Plateau v116 可退订"];
  gapUnsub["Gap: 邀请邮件无法退订"];
  wpUnsub["WP-email-invite-unsubscribe"];
  taskFE --> gw;
  gw --> auth;
  auth --> unsub;
  auth --> evUnsub;
  ten --> auth;
  ev --> auth;
  p115 --> gapUnsub;
  wpUnsub --|> gapUnsub;
  wpUnsub --|> p116;
```
