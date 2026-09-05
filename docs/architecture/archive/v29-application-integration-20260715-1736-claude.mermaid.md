# Application Integration v29 — taskGitOauth Go Migration

```mermaid
flowchart LR
  Vue[Vue git-site-oauth]
  GW[task-gateway]
  GO["🟢 taskGitOauth :8002"]
  Django[saas-backend internal/bind]
  Cred[taskCredentialService]
  Proj[taskProjectService]
  Cloud[taskCloudService]
  DB[(db/git-oauth SQLite)]
  GH[GitHub OAuth]
  GL[GitLab OAuth]
  PY["[DEPRECATED] Django gitOauth"]

  Vue --> GW --> GO
  GO --> GH
  GO --> GL
  GO --> Django
  GO --> DB
  Cred --> GO
  Proj --> GO
  Cloud --> GO
  PY -.->|replaced by| GO
```

## Plateau / Gap

- **Plateau v27 ✅ current**: 终态硬释放基线（并行 v28 nested-git-repos）
- **Gap closed**: gitOauth 仍为 Django，与 Go-first 元规则及运维目标冲突；删除 Python 前置未满足
- **Plateau v29 🎯 target**: `taskGitOauth` Go 直替 :8002；同库同契约；清理 Python `gitOauth`
