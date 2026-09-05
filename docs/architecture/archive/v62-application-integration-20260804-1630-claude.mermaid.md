# v62 — RBAC 含小组与资源分配

> ArchiMate Mermaid | v62 target | rbac-groups | claude | 2026-08-04

```mermaid
graph TD;
  taskAuth["taskAuth PDP [MODIFIED v62]"];
  taskTenant["taskTenantService [MODIFIED v62]"];
  taskProject["taskProjectService [MODIFIED v62]"];
  taskCloud["taskCloudService [MODIFIED v62]"];
  shareLibAuthz["shareLib/authz Thin Client [MODIFIED v62]"];
  gateway["APISIX Gateway"];
  kafkaBroker["Kafka"];
  taskAuthDB["taskAuth DB"];
  taskTenantDB["taskTenant DB (+group_admin +resource_assignment)"];
  roleGroupAdmin["group_admin (p=75) [NEW v62]"];
  roleTenantAdmin["tenant_admin (p=100)"];
  roleMember["member (p=50)"];
  plateauV61["Plateau v61: RBAC baseline"];
  plateauV62["Plateau v62: +group roles +resource assignment"];
  gapGroups["Gap: no group_admin, no resource-group"];
  wpGroups["WP: group_admin + resource-group + API (3d)"];

  gateway --> taskAuth;
  gateway --> taskTenant;
  shareLibAuthz -->|"+RequireGroupAdmin"| taskTenant;
  shareLibAuthz -->|"+HasGroupResourceAccess"| taskProject;
  taskAuth ..-> taskAuthDB;
  taskTenant ..-> taskTenantDB;
  roleTenantAdmin --*|"管理"| roleGroupAdmin;
  roleGroupAdmin --*|"管理组内成员"| roleMember;
  plateauV61 --- gapGroups;
  wpGroups --|> gapGroups;
  wpGroups --|> plateauV62;

  style roleGroupAdmin fill:#LightGreen,stroke:#333
  style taskTenant fill:#LightYellow,stroke:#333
  style taskProject fill:#LightYellow,stroke:#333
  style taskTenantDB fill:#LightYellow,stroke:#333
```

## Changes (v61 → v62)

| Mark | Component | Detail |
|------|-----------|--------|
| 🟢 NEW | group_admin role | priority=75, scope=group_id, manages group members |
| 🟢 NEW | tenant_group_admin table | who manages which group |
| 🟢 NEW | tenant_resource_group_assignment | resource→group mapping |
| 🟢 NEW | RequireGroupAdmin middleware | scoped admin check |
| 🟡 MOD | tenant_group_member handlers | requireCompanyMember → RequireGroupAdmin |
| 🟡 MOD | taskProject/taskCloud | + group resource inheritance check |
| 🟡 MOD | taskTenantDB | +2 tables |
