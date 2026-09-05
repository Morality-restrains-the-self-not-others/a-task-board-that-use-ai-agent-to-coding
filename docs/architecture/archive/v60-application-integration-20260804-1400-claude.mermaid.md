# v60 — RBAC 集中授权架构

> ArchiMate Mermaid | v60 target | rbac-v3 | claude | 2026-08-04

```mermaid
graph TD;
  taskAuth["taskAuth PDP (:8003) [MODIFIED v60]"];
  taskTenant["taskTenantService (:8020) [MODIFIED v60]"];
  taskCloud["taskCloudService (:8018) [MODIFIED v60]"];
  taskBill["taskBill (:8004) [MODIFIED v60]"];
  taskEvents["taskEvents (:8023) [MODIFIED v60]"];
  shareLibAuthz["shareLib/authz Thin Client [REFINED v60]"];
  gateway["APISIX Gateway [MODIFIED v60]"];
  kafkaBroker["Kafka"];
  kafkaConsumers["Kafka Consumers"];
  taskAuthDB["taskAuth DB (+4 RBAC tables)"];
  taskTenantDB["taskTenant DB (+2 RBAC tables)"];
  evtRoleAssigned["PlatformRoleAssigned"];
  evtRoleRevoked["PlatformRoleRevoked"];
  evtTenantRole["TenantMemberRoleChanged"];
  evtGroupRole["TenantGroupRoleChanged [NEW v60]"];
  evtCompanyCreated["CompanyCreated (existing)"];

  %% Gateway → PDP (centralized auth)
  gateway -->|"forward-auth → PDP check"| taskAuth;
  gateway --> taskTenant;
  gateway --> taskCloud;
  gateway --> taskBill;

  %% Thin client serves services (read Context, fallback HTTP to PDP)
  shareLibAuthz -->|"CheckPermission HTTP fallback"| taskAuth;
  shareLibAuthz -->|"RequireRole/RequireTenantRole"| taskTenant;
  shareLibAuthz -->|"RequireRole"| taskCloud;
  shareLibAuthz -->|"RequireRole"| taskBill;

  %% Kafka flow
  kafkaBroker --> kafkaConsumers;
  kafkaConsumers -->|"CompanyCreated→assign role [NEW]"| taskTenant;
  kafkaConsumers --> taskAuth;
  kafkaConsumers -->|"RoleChanged→cache invalidation [NEW]"| taskEvents;

  %% Data ownership
  taskAuth ..-> taskAuthDB;
  taskTenant ..-> taskTenantDB;

  %% New events
  taskAuth --> evtRoleAssigned --> kafkaBroker;
  taskAuth --> evtRoleRevoked --> kafkaBroker;
  taskTenant --> evtTenantRole --> kafkaBroker;
  taskTenant --> evtGroupRole --> kafkaBroker;
  evtCompanyCreated --> kafkaBroker;

  style taskAuth fill:#LightYellow,stroke:#333
  style shareLibAuthz fill:#LightGreen,stroke:#333
  style evtGroupRole fill:#LightGreen,stroke:#333
```

## Key Changes (v60 vs v59)

| Change | Detail |
|--------|--------|
| 🔄 REFINED | shareLib/authz: heavy engine → thin client (Context read O(1) + HTTP fallback to PDP) |
| 🟢 NEW | taskAuth PDP: `/api/internal/authz/check` centralized decision point |
| 🟢 NEW | tenant_group_role table: group → role inheritance |
| 🟢 NEW | CompanyCreated consumer: auto-assign tenant_admin to creator |
| 🟢 NEW | TenantGroupRoleChanged event |
| 🟡 MODIFIED | Phase 1: 8-14 days → 5-7 days (simplified scope) |
