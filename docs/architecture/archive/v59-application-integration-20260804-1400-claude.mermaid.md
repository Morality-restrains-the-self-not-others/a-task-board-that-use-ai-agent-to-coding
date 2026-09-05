# v59 Application Integration — 身份角色权限体系

> ArchiMate Mermaid 渲染 | 版本: v59 target | 迭代: identity-role-permission-system | 作者: claude | 2026-08-04 14:00

```mermaid
graph TD;
  taskAuth["taskAuth (:8003) [MODIFIED v59]"];
  taskGitOauth["taskGitOauth (:8002)"];
  taskTenant["taskTenantService (:8020) [MODIFIED v59]"];
  taskProject["taskProjectService (:8016)"];
  taskTask["taskTaskService (:8021)"];
  taskAIComment["taskAIComment (:8022)"];
  taskCloud["taskCloudService (:8018)"];
  taskBill["taskBill (:8004)"];
  taskEvents["taskEvents (:8023)"];
  shareLibAuthz["shareLib/authz Permission Middleware [NEW v59]"];
  kafkaConsumers["Kafka Consumers (Go)"];
  gateway["APISIX Gateway (:18081) [MODIFIED v59]"];
  kafkaBroker["Kafka Domain Events Bus"];
  taskAuthDB["taskAuth DB (+roles/permissions) [MODIFIED v59]"];
  taskGitOauthDB["taskGitOauth DB"];
  taskTenantDB["taskTenant DB (+member_roles) [MODIFIED v59]"];
  taskProjectDB["taskProject DB"];
  taskTaskDB["taskTask DB"];
  taskCloudDB["taskCloud DB"];
  taskBillDB["taskBill DB"];
  evtRoleAssigned["PlatformRoleAssigned [NEW v59]"];
  evtRoleRevoked["PlatformRoleRevoked [NEW v59]"];
  evtTenantRole["TenantRoleChanged [NEW v59]"];
  oldIsAdmin["is_admin direct check [DEPRECATED v59]"];
  oldIsSuperuser["is_superuser direct check [DEPRECATED v59]"];

  %% Gateway routing
  gateway --> taskAuth;
  gateway --> taskTenant;
  gateway --> taskProject;
  gateway --> taskTask;
  gateway --> taskCloud;
  gateway --> taskBill;
  gateway -->|"forward-auth → roles"| shareLibAuthz;

  %% Permission middleware → services
  shareLibAuthz -->|"platform role check"| taskAuth;
  shareLibAuthz -->|"tenant role check"| taskTenant;
  shareLibAuthz -->|"platform role check"| taskCloud;
  shareLibAuthz -->|"platform role check"| taskBill;
  shareLibAuthz -->|"tenant role check"| taskProject;
  shareLibAuthz -->|"tenant role check"| taskTask;

  %% Kafka event flow
  kafkaBroker --> kafkaConsumers;
  kafkaConsumers --> taskTenant;
  kafkaConsumers --> taskAuth;
  kafkaConsumers --> taskProject;
  kafkaConsumers --> taskCloud;
  kafkaConsumers --> taskEvents;

  %% Data ownership
  taskAuth ..-> taskAuthDB;
  taskTenant ..-> taskTenantDB;
  taskProject ..-> taskProjectDB;
  taskTask ..-> taskTaskDB;
  taskCloud ..-> taskCloudDB;
  taskBill ..-> taskBillDB;

  %% New domain events
  taskAuth --> evtRoleAssigned;
  taskAuth --> evtRoleRevoked;
  taskTenant --> evtTenantRole;
  evtRoleAssigned --> kafkaBroker;
  evtRoleRevoked --> kafkaBroker;
  evtTenantRole --> kafkaBroker;

  %% Deprecated patterns
  oldIsAdmin -.->|"[DEPRECATED]"| taskTenant;
  oldIsSuperuser -.->|"[DEPRECATED]"| taskAuth;

  %% Styling
  style shareLibAuthz fill:#LightGreen,stroke:#333;
  style evtRoleAssigned fill:#LightGreen,stroke:#333;
  style evtRoleRevoked fill:#LightGreen,stroke:#333;
  style evtTenantRole fill:#LightGreen,stroke:#333;
  style taskAuth fill:#LightYellow,stroke:#333;
  style taskTenant fill:#LightYellow,stroke:#333;
  style gateway fill:#LightYellow,stroke:#333;
  style taskAuthDB fill:#LightYellow,stroke:#333;
  style taskTenantDB fill:#LightYellow,stroke:#333;
  style oldIsAdmin fill:#LightPink,stroke:#333;
  style oldIsSuperuser fill:#LightPink,stroke:#333;
```

## 变更图例

| 标记 | 含义 | 元素 |
|------|------|------|
| 🟢 [NEW v59] | 本次新增 | shareLibAuthz, evtRoleAssigned, evtRoleRevoked, evtTenantRole |
| 🟡 [MODIFIED v59] | 本次修改 | taskAuth, taskTenant, gateway, taskAuthDB, taskTenantDB |
| 🔴 [DEPRECATED v59] | 本次废弃 | oldIsAdmin, oldIsSuperuser |

## 关键数据流

1. **Gateway → shareLibAuthz**: forward-auth 注入 X-User-Roles + X-User-Permissions
2. **shareLibAuthz → Services**: 统一权限检查中间件（platform/tenant role check）
3. **taskAuth → Kafka**: 平台角色变更事件（PlatformRoleAssigned/Revoked）
4. **taskTenant → Kafka**: 租户角色变更事件（TenantRoleChanged）
