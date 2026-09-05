# GitLab taskAuth SSO 跨子域重复登录

## 背景

用户已在 `https://www.daydaymoney.com` 登录，从项目页打开「代码仓库」→ GitLab → 点击 taskAuth SSO 后，仍被带到：

`https://www.daydaymoney.com/auth/login/?next=https://api.daydaymoney.com/api/oidc/authorize?...`

表现为「已登录却再次要求登录」。

## 目标

已登录主站的用户，经 GitLab「taskAuth SSO」完成 OIDC 授权并回到 GitLab 会话，**不再**因跨子域 Cookie / gateway base 误判而停在登录页。

## 根因（2026-07-12）

1. **构建期把 `VITE_TASK_GATEWAY_PUBLIC_BASE` 烤成空串**（复用了同源空 `apiBaseUrl`）。公网 SPA 在 `www` 上把 gateway 误判为 `www:18081`，`sanitizeOidcResumeNext` 拒绝 `api.daydaymoney.com` 的 `next`，已登录 resume 静默失败。
2. **`userId` / `sessionid` Cookie 为 host-only（仅 www）**。顶层导航到 `api.daydaymoney.com/api/oidc/authorize` 时浏览器不携带该 Cookie，taskAuth 判定未登录并 302 回登录页。

## 方案

| 项 | 做法 |
|----|------|
| Gateway base | Vite 烘焙 `gatewayOrigin`（`https://api.${baseDomain}`）；运行时若仍为空则从 `www.*` 推导 `api.*` |
| Cookie Domain | 生产写入 `Domain=.${baseDomain}; SameSite=Lax; Secure`（IP/localhost 仍 host-only） |
| OIDC resume | Login 续接前重写 `userId` 共享 Cookie |
| Authorize 未登录跳转 | taskAuth 使用 confload 解析后的 `postLogoutRedirectOrigins` 中 www 入口 |
| confload | `resolveStructStrings` 须解析 `[]string`（如 postLogoutRedirectOrigins） |

## 验收

- 已登录用户打开带 `next=https://api.../api/oidc/authorize` 的登录 URL → 自动续接 authorize，不停留在登录表单。
- authorize 请求携带 `.example.com` 的 `userId`（或等价会话）→ 302 带 `code=` 回 GitLab callback。
- 单元测试：前端 `oidcResumeUrl` / `ssoCookieDomain`；Django cookie domain；taskAuth www login base；**confload string slice**。
- E2E：`Gitlab.daydaymoney-cross-subdomain-sso`（真实 www+api+gitlab 公网子域）。



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：跨子域 Cookie/登录修复，无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| GitLab taskAuth SSO 跨子域重复登录 | — | — | — | — | 跨子域 Cookie/登录修复，无新增业务事件 |
## 变更记录

- 2026-07-12：初版，修复 www/api 跨子域 SSO 重复登录。
- 2026-07-12：修好 confload `[]string` 模板解析；去掉 gateway→www 兜底；补公网跨子域 Playwright。
