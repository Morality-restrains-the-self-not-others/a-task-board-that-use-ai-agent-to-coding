# v108 application-integration — 超管微信分账与动账通知 (current)

```mermaid
graph TD;
  staff["平台员工"];
  taskFE["taskFE SPA 🟡 MODIFIED v108"];
  gw["taskGateway / APISIX 🟡 MODIFIED v108"];
  bill["taskBill 🟡 MODIFIED v108"];
  wx["微信支付 APIv3"];
  ps["billing_profit_sharing 🟡"];
  audit["billing_profit_sharing_admin_action 🟢"];
  inbox["billing_profit_sharing_change_notify 🟢"];
  plateauV107["Plateau v107"];
  plateauV108["Plateau v108 超管分账与动账通知"];
  gapPs["Gap: 超管无微信单号、无带审计手工分账、动账通知未解密"];
  wpPs["WP-admin-profit-sharing-change-notify"];
  staff --> taskFE;
  taskFE --> gw;
  gw --> bill;
  wx --> gw;
  bill --> wx;
  bill --> ps;
  bill --> audit;
  bill --> inbox;
  plateauV107 --> gapPs;
  wpPs --|> gapPs;
  wpPs --|> plateauV108;
```
