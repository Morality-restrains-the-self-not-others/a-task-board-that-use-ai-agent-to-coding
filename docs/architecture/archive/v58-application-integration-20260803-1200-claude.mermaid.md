# v58 Application Integration — 邮箱邀请增强

> 🟡 [MODIFIED] taskAuth — 新增 invite_reason / account_expires_at / assigned_role 字段 + GET list 返回 inviterName

```mermaid
graph TD;
  app-taskAuth["taskAuth (:8003) 🟡 MODIFIED"];
  app-taskGitOauth["taskGitOauth (:8002)"];
  app-taskTenant["taskTenantService (:8020)"];
  app-taskProject["taskProjectService (:8016)"];
  app-taskTask["taskTaskService (:8021)"];
  app-taskAIComment["taskAIComment (:8022)"];
  app-taskCloud["taskCloudService (:8018)"];
  app-taskBill["taskBill (:8004)"];
  app-taskEvents["taskEvents (:8023)"];
  app-kafkaConsumers["Kafka Consumers (Go)"];
  app-gateway["APISIX Gateway (:18081)"];
  app-djangoRemoved["Django saas-backend 🔴 REMOVED v57"];
  data-kafkaBroker["Kafka Domain Events Bus"];
  data-taskAuthDB["taskAuth DB"];
  data-taskGitOauthDB["taskGitOauth DB"];
  data-taskTenantDB["taskTenant DB"];
  data-taskProjectDB["taskProject DB"];
  data-taskTaskDB["taskTask DB"];
  data-taskCloudDB["taskCloud DB"];
  data-taskBillDB["taskBill DB"];

  app-gateway --> app-taskAuth;
  app-gateway --> app-taskGitOauth;
  app-gateway --> app-taskTenant;
  app-gateway --> app-taskProject;
  app-gateway --> app-taskTask;
  app-gateway --> app-taskAIComment;
  app-gateway --> app-taskCloud;
  app-gateway --> app-taskBill;
  data-kafkaBroker --> app-kafkaConsumers;
  app-kafkaConsumers --> app-taskTenant;
  app-kafkaConsumers --> app-taskAuth;
  app-kafkaConsumers --> app-taskProject;
  app-kafkaConsumers --> app-taskCloud;
  app-kafkaConsumers --> app-taskEvents;
  app-taskAuth ..-> data-taskAuthDB;
  app-taskGitOauth ..-> data-taskGitOauthDB;
  app-taskTenant ..-> data-taskTenantDB;
  app-taskProject ..-> data-taskProjectDB;
  app-taskTask ..-> data-taskTaskDB;
  app-taskCloud ..-> data-taskCloudDB;
  app-taskBill ..-> data-taskBillDB;
```

## 变更说明

- **版本**: v58 🎯 target
- **迭代**: email-invite-enhancement
- **日期**: 2026-08-03 12:00
- **变更**: taskAuth 组件修改 — 邮箱邀请新增 3 个字段 + 邀请人姓名
- **基于**: v57 (v57-application-integration-20260730-0229-claude.puml)
