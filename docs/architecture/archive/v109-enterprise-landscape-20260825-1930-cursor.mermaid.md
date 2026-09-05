# v109 enterprise-landscape — 系统管理租户详情 (current)

```mermaid
graph TD;
  staff["平台员工 / 超管"];
  fe["taskFE 系统管理 🟡"];
  tenant["taskTenantService 🟡"];
  bill["taskBill 🟡"];
  proj["taskProjectService 🟡"];
  p108["Plateau v108"];
  p109["Plateau v109"];
  g["Gap: 租户名纯文本，无详情页"];
  wp["WP-system-admin-tenant-detail"];
  staff --> fe;
  fe --> tenant;
  fe --> bill;
  fe --> proj;
  p108 --> g;
  wp --|> g;
  wp --|> p109;
```
