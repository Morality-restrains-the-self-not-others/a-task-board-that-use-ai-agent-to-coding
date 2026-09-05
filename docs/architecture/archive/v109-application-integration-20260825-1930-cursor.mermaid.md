# v109 application-integration — 系统管理租户详情 (current)

```mermaid
graph TD;
  staff["平台员工"];
  taskFE["taskFE SPA 🟡 MODIFIED v109"];
  gw["taskGateway / APISIX 🟡 MODIFIED v109"];
  tenant["taskTenantService 🟡"];
  bill["taskBill 🟡"];
  proj["taskProjectService 🟡"];
  co["tenant_company"];
  quota["billing quotas"];
  ws["project_workspace_entries"];
  ord["billing_resource_order"];
  plateauV108["Plateau v108"];
  plateauV109["Plateau v109 租户详情"];
  gapTd["Gap: 租户名不可点"];
  wpTd["WP-system-admin-tenant-detail"];
  staff --> taskFE;
  taskFE --> gw;
  gw --> tenant;
  gw --> bill;
  gw --> proj;
  tenant --> co;
  bill --> quota;
  bill --> ord;
  proj --> ws;
  plateauV108 --> gapTd;
  wpTd --|> gapTd;
  wpTd --|> plateauV109;
```
