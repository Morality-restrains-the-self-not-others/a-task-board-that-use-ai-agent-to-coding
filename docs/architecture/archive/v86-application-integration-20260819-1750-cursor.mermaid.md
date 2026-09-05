# v86 — Personal account deletion (PIPL)

```mermaid
flowchart LR
  FE[taskFE UserProfile 危险区 MOD] -->|precheck / 提交注销 / 冷却撤回| Auth[taskAuth 自助注销 API + precheck 聚合 MOD]
  Auth -->|internal GET account-deletion-blockers| Bill[taskBill internal blockers MOD]
  Auth -->|internal GET account-deletion-blockers| Tenant[taskTenantService internal blockers MOD]
  Auth -->|internal GET account-deletion-blockers| Cloud[taskCloudService internal blockers MOD]
  Auth -->|写请求/状态/冷却| Req[(auth_account_deletion_request NEW)]
  Auth -->|注销请求已受理| Ev[AccountDeletionRequested Kafka NEW]
  Timer[taskEvents timer intent account-deletion-execute-due NEW] -->|POST execute-due 幂等触发| Auth
  Auth -->|execute 归档/脱敏/通知| Archive[(归档 + 脱敏)]
```

- **Version:** v86 target
- **Iteration:** user-account-deletion-pipl-v86
- **Based on:** v85
- **Decisions:** 个人账号注销自助入口（PIPL）；冷却期可撤回；物理 DELETE 不做（归档+脱敏）；超管 archive 路径保留人工兜底
- **Events:** Kafka 五事件（AccountDeletionRequested / 各阶段状态）跨服务编排；execute-due 由 taskEvents timer worker 幂等触发
