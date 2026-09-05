# Application Integration v32 — taskAiProvider Go Migration

```mermaid
flowchart LR
  Vue[Vue SPA vendor/admin]
  GW[task-gateway]
  GO["🟢 taskAiProvider :8010"]
  Auth[taskAuth OIDC]
  Cloud[taskCloudService :8018]
  SaaS[saas-backend SSO bridge]
  DB[(db/ai-provider SQLite)]
  OCI[OCI Registry]
  PY["[DEPRECATED] Django Saas_Ai_Provider"]

  Vue --> GO
  GW --> GO
  GO --> Auth
  SaaS --> GO
  GO --> Cloud
  Cloud --> GO
  GO --> DB
  GO --> OCI
  PY -.->|replaced by| GO
```

## Plateau / Gap

- **Plateau v31 ✅ current**: OTP/SMS 全切 Go（taskAuth）
- **Gap closed**: ai-provider 仍为 Django，与 Go-first 元规则及运维目标冲突；删除 Python 前置未满足；go-split Non-Goal 废止
- **Plateau v32 🎯 target**: `taskAiProvider` Go 直替 :8010；同库同契约；清理 Python `Saas_Ai_Provider`
