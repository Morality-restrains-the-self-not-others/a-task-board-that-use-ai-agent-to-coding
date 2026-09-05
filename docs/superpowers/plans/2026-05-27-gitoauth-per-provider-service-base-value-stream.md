# Value Stream: gitOauth 内部 API 按 service_provider 路由

> Derived from design: `docs/superpowers/specs/2026-05-27-gitoauth-per-provider-service-base-design.md`

## Value Summary

用户在 git-site-oauth 页切换不同 Git 站点 tab 时，绑定状态查询应命中各自 `service.allowedHost`，本地 `gitlab-local` 不因公网 gitOauth 502 而失败。

## End-to-End Flow

[用户切换 provider tab] → [GET connection?service_provider=…] → [主站 resolve service_base] → [gitOauth internal summary] → [200/503 仅影响当前 tab]

## Value Increments

### Increment 1: per-provider summary 路由（薄切片）
**Value to user:** gitlab-local tab 可加载绑定状态  
**Scope:** `resolve_gitoauth_service_base` + `fetch_gitoauth_provider_credential_summary_for_user`  
**Depends on:** nothing

### Increment 2: 全 internal API 对齐
**Value to user:** 换票/delete/audit 与 start 路由一致  
**Scope:** `github_app_tokens.py` 其余函数 + 移除 `DJANGO_GITOAUTH_BASE`

### Increment 3: Gitlab delete provider_key 修补
**Value to user:** gitlab-local 解绑正确  
**Scope:** `GitlabAppConnectionView.delete`
