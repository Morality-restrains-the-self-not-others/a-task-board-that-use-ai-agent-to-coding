# Application Integration v18 — auto_run 首指令与自动交付

```mermaid
flowchart LR
  Vue[Vue Frontend] -->|create auto_run| GW[task-gateway]
  GW --> TTS[taskTaskService]
  TTS -->|start-vm| Cloud[taskCloudService]
  Cloud --> OSJS[onlineServiceJS]
  OSJS -->|task-detail auto_run + identities| CRED[taskCredentialService]
  OSJS -->|AutoRunFirstInstruction| Job[createJob trae]
  Job -->|completed| Del[AutoRunDelivery]
  Del -->|sync identities| OSJS
  Del -->|commit| Git[(layer git)]
  Del -->|oauth-refresh-push| PR[GitHub/GitLab PR]
```

## 变更摘要

- 🟡 taskCredentialService：`auto_run`、`repo_git_identities`
- 🟡 onlineServiceJS：首指令 + 完成后交付
- 🟢 AutoRunFirstInstruction / AutoRunDelivery 应用功能
