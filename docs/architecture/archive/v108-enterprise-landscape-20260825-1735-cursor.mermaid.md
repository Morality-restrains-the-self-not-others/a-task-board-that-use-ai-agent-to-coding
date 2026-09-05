# v108 enterprise-landscape — 超管微信分账与动账通知 (current)

```mermaid
graph TD;
  staff["平台员工 / 超管"];
  wxMch["微信支付商户后台"];
  recon["分账对账与补分账 🟡"];
  notifyCfg["配置动账通知 URL 🟢"];
  fe["taskFE 系统管理 🟡"];
  bill["taskBill 🟡"];
  p107["Plateau v107"];
  p108["Plateau v108"];
  g["Gap: 超管无法对照微信单号并带审计补分账"];
  wp["WP-admin-profit-sharing-change-notify"];
  staff --> recon;
  staff --> fe;
  fe --> bill;
  wxMch --> notifyCfg;
  wxMch --> bill;
  p107 --> g;
  wp --|> g;
  wp --|> p108;
```
