# Value Stream: relay 直启预检内网 origin 修复

> Derived from design: `docs/superpowers/specs/2026-05-27-relay-precheck-local-origin-design.md`

## Value Summary

本地开发直启 relay 时，用户在 OAuth 已授权情况下不再因预检误打公网网关（502）而看到误导性「完成 OAuth 绑定」提示；预检走内网 Django，409 与 502 错误引导分流。

## End-to-End Flow

[用户点击直启「启动」] → [token-init 本地签发 container token] → [precheck 走 internalApiBase 调 repo-clone-credentials] → [200 继续 start / 409 展示缺失仓库 / 5xx 展示连通性错误] → [关联项目展示 OAuth + 克隆账号已保存双状态]

## Value Increments

### Increment 1: 预检内网 origin 薄切片（Thin Slice）
**Value to user:** 本地直启预检不再 502，OAuth 已授权且账号已保存时可启动。  
**Scope:** `relay_to_trae_repo_credentials_precheck` 使用 `get_internal_task_api_base_url()`。  
**Depends on:** 无。

### Increment 2: 预检错误引导分流（Core Value）
**Value to user:** 502 不再提示 OAuth 绑定；409 仍展示保存账号引导。  
**Scope:** `ServerConfig.logic.vue` 按 status/error_code 分流文案与 guide 可见性。  
**Depends on:** Increment 1。

### Increment 3: 克隆账号已保存徽章（Enhancement）
**Value to user:** 关联项目区分 OAuth 与 per-repo 账号保存状态。  
**Scope:** `TaskDetailLinkedProjectsPanel.vue` + 单测。  
**Depends on:** Increment 2。

### Increment 4: 回归保护（Essential Support）
**Value to user:** 行为长期稳定。  
**Scope:** pytest + playwright 扩展。  
**Depends on:** Increment 1–2。

## Affected Existing Streams

- `task-detail-oauth-binding-guidance`
- `task-detail-repo-clone-credentials-contract`
- `relay-token-audit-observability`
