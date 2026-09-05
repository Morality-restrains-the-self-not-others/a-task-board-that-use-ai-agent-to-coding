# v61 — RBAC 角色权限体系

> ArchiMate Mermaid | v61 target | rbac | claude | 2026-08-04

```mermaid
graph TD;
  taskAuth["taskAuth (:8003) [MODIFIED v61]"];
  taskTenant["taskTenantService (:8020) [MODIFIED v61]"];
  taskCloud["taskCloudService (:8018) [MODIFIED v61]"];
  taskBill["taskBill (:8004) [MODIFIED v61]"];
  taskEvents["taskEvents (:8023) [MODIFIED v61]"];
  shareLibAuthz["shareLib/authz Thin Client [NEW v61]"];
  gateway["APISIX Gateway [MODIFIED v61]"];
  kafkaBroker["Kafka"];
  taskAuthDB["taskAuth DB (+4 RBAC tables)"];
  taskTenantDB["taskTenant DB (+2 RBAC tables)"];
  evtRoleAssigned["PlatformRoleAssigned"];
  evtTenantRole["TenantMemberRoleChanged"];
  evtGroupRole["TenantGroupRoleChanged"];
  plateauV57["Plateau v57"];
  plateauV61["Plateau v61: RBAC"];
  gapRBAC["Gap: no RBAC"];
  wpPhase1["WP Phase1 (7d)"];

  gateway -->|"forward-auth → PDP"| taskAuth;
  gateway --> taskTenant;
  gateway --> taskCloud;
  gateway --> taskBill;
  shareLibAuthz -->|"RequireRole"| taskAuth;
  shareLibAuthz -->|"RequireTenantRole"| taskTenant;
  shareLibAuthz -->|"RequireRole"| taskCloud;
  shareLibAuthz -->|"RequireRole"| taskBill;
  taskAuth ..-> taskAuthDB;
  taskTenant ..-> taskTenantDB;
  taskAuth --> evtRoleAssigned --> kafkaBroker;
  taskTenant --> evtTenantRole --> kafkaBroker;
  taskTenant --> evtGroupRole --> kafkaBroker;
  plateauV57 --- gapRBAC;
  wpPhase1 --|> gapRBAC;
  wpPhase1 --|> plateauV61;

  style shareLibAuthz fill:#LightGreen,stroke:#333
  style taskAuth fill:#LightYellow,stroke:#333
  style taskTenant fill:#LightYellow,stroke:#333
```

## Changes (v57 → v61)

| Mark | Component | Detail |
|------|-----------|--------|
| 🟢 NEW | shareLib/authz | Thin client: RequireRole/RequireTenantRole |
| 🟡 MOD | taskAuth | +4 RBAC tables + role APIs + PDP |
| 🟡 MOD | taskTenantService | +2 RBAC tables + member/group role APIs |
| 🟡 MOD | taskCloud/Bill | hand-rolled auth → authz |
| 🟡 MOD | APISIX | forward-auth injects X-User-Roles |
| 🔴 DEPR | is_admin/is_superuser | Direct checks → role-based |
