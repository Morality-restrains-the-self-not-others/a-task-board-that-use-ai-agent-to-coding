# v77 — Member Joined Auto Git Identity

```mermaid
flowchart LR
  TT[taskTenantService MOD] -->|publish MEMBER_JOINED| K[Kafka member-joined MOD]
  K --> C[1_create_default_git_identity NEW]
  C -->|POST ensure-default| TTS[taskTaskService MOD]
  TTS --> TGI[(task_git_identities)]
  FE[taskFE PeopleManage MOD] -->|tenant manage APIs| TTS
```

- **Version:** v77 target
- **Iteration:** member-joined-auto-git-identity-v77
- **Based on:** v76
- **Decisions:** 复用 MEMBER_JOINED；默认邮箱 `sha256` 哈希 + `@daydaymoney.com`；鉴权 self / `member:manage` / `group-members:manage`
