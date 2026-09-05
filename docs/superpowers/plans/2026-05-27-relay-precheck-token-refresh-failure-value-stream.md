# Value Stream: relay 预检 token 换发失败分流

**设计：** `docs/superpowers/specs/2026-05-27-relay-precheck-token-refresh-failure-design.md`

## 增量

### VS-1 后端凭证构建诊断（必做）
- 扩展 `_build_repo_clone_credentials` 返回 `token_refresh_failures` / `missing_identity_repos`
- `repo-clone-credentials` 409 vs 502 分流
- relay precheck 透传

**Value to user:** 预检不再把 GitLab 宕机误报为「缺 OAuth」。

### VS-2 前端预检引导分流（必做）
- `ServerConfig.logic.vue` 按 error_code 展示文案
- amber OAuth 引导仅在真正缺 identity 时出现

### VS-3 relay 直启克隆身份 UI（必做）
- `relayToTrae` 路由下展示克隆身份选择器

**Scope:** `task-detail-oauth-binding-guidance`, `task-detail-repo-clone-credentials-contract`, `relay-precheck-internal-origin`
