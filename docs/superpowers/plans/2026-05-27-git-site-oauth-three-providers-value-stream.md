# Value Stream: Git 网站授权页展示全部 service_provider

> Derived from design: `docs/superpowers/specs/2026-05-27-git-site-oauth-three-providers-design.md`

## Value Summary

用户在 Git 网站授权设置页能看到与 `port_config.json` 一致的 **全部 OAuth 站点**（3 个 `service_provider`），并可分别查看绑定状态、发起授权。

## End-to-End Flow

[用户打开 git-site-oauth] → [GET providers 目录] → [渲染 N 个站点切换按钮] → [选中站点 + service_provider] → [GET user-scoped connection?service_provider=…] → [展示绑定/授权] → [start 携带 service_provider 跳转 gitOauth]

## Value Increments

### Increment 1: 多站点目录与 UI（Thin Slice）

**Value to user:** 页面展示 3 个站点切换项，与配置一致  
**Scope:** `GET /api/accounts/git-oauth/providers/` + `UserGitSiteOAuthSettings.vue` 动态选项 + Playwright 计数断言  
**Depends on:** nothing

### Increment 2: 按 service_provider 查询与授权

**Value to user:** 切换站点后 connection/start 命中正确 `provider_key`  
**Scope:** connection/start 查询参数；`GithubAppAuthorizeStartView` JWT 携带 `service_provider`  
**Depends on:** Increment 1
