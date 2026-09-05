# [运行时] GitHub OAuth 授权回调 404 page not found（8c47b8a 路由前缀错配回归）

## 基本信息

- 案例编号：OPT-20260807-020-OAUTH-CALLBACK-404（2026-08-07 修复）
- 关联服务：taskGitOauth（:8002）、边缘 nginx、GitHub App 白名单、conf/base.yaml SSOT
- 影响链路：项目页「OAuth 授权」→ GitHub authorize → 回调 → **404**，授权流程中断

## 失败现象

- 页面 `/tenant/{id}/projects/{proj}/` 点击「OAuth 授权」→ GitHub 授权页正常 → 授权后
  跳回 `https://gitoauth_api.daydaymoney.com/api/accounts/github/oauth/callback/?code=...&state=...`
  显示 **404 page not found**（`content-type: text/plain`，19 字节 —— Go `http.ServeMux` 默认 404）。
- `/api/health/` 正常 200（服务存活、nginx 反代正常），迷惑性强。

## 根因（回归）

`8c47b8a`（2026-08-04）flatten 重构把浏览器回调入口 `handleAccountsDynamic` 的注册前缀
从 `/api/accounts/` **错改为 `/api/git-oauth/accounts/`**，而该 handler 内部只解析
`/api/accounts/<service_provider>/oauth/callback/`（app.go L136 注释即契约路径）。
回调 URL 由 **SSOT redirect_uri**（`conf/base.yaml` `subdomains.gitoauth` + provider
`redirect_uri` 模板）与 **GitHub/GitLab App 白名单** 固定，无法随重构扁平化 ——
契约路径与注册路径错配 → 回调永远 404。

## 诊断线索

- 404 响应带 `traceparent`/`x-trace-id`（应用中间件已加）→ 请求已到 taskGitOauth，非网关/nginx 层
- `curl -sSI https://gitoauth_api.daydaymoney.com/api/health/` = 200 但 callback = 404
- `git log` 定位 8c47b8a 为唯一路由注册变更（Red 测试：恢复旧注册前新测试失败）

## 修复

- `src/app.go`：恢复 `mux.HandleFunc("/api/accounts/", a.handleAccountsDynamic)`（契约路径），
  删除无调用方的 `/api/git-oauth/accounts/` 注册；注释标明该路径**不可扁平化**
- 回归测试：`TestAccountsCallbackRouteRegisteredContractPath`（github/gitlab 契约路径非 404 且 302）
  + `TestFlattenedCallbackRoutesStillRegistered`（扁平路径防回归）
- 验证：`go test ./...` 全绿；重启后公网
  `gitoauth_api.daydaymoney.com/api/accounts/github/oauth/callback/` = **302**（带 trace_id），不再 404

## 防再发

- **浏览器回调契约路径（OAuth 提供商白名单决定）禁止参与内部路由扁平化重构**；
  凡涉及 `redirect_uri`/回调路径的重构，必须先用契约路径路由级测试锁定（RegisterRoutes + httptest）
- 重构删除/改名路由时，grep 全仓契约路径（`/api/accounts/`、`redirect_uri` 模板）确认调用面
- 伴读：`conf/auth/git-oauth/ai.md`、失败经验 80（GitLab redirect_uri mismatch 姊妹案例）
