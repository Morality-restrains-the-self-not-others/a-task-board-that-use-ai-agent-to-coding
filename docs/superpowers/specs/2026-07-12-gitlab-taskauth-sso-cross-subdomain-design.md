# GitLab taskAuth SSO 跨子域重复登录 — 设计与修复

- **日期**: 2026-07-12
- **迭代**: gitlab-taskauth-sso-cross-subdomain
- **状态**: 已实现

## 问题现象

主站已登录 → GitLab taskAuth SSO → 落回
`https://www.daydaymoney.com/auth/login/?next=https://api.daydaymoney.com/api/oidc/authorize?...`

## 根因

1. Vite 将 `VITE_TASK_GATEWAY_PUBLIC_BASE` 错误地设为与空 `apiBaseUrl` 相同，生产包无法识别 `api` 域 `next`。
2. SSO 桥接 Cookie（`userId`）写在 `www` host-only，顶层跳到 `api` authorize 时无凭证。
3. Go confload `resolveStructStrings` 未处理 `[]string`，`postLogoutRedirectOrigins` 残留 `${scheme}://...`。

## 方案对比

| 方案 | 做法 | 结论 |
|------|------|------|
| **A（采用）** | Cookie `Domain=.baseDomain` + 正确烘焙/推导 gateway base + 修好 confload | 与现有 userId 桥一致 |
| B | www 侧 BFF 换票再 302 | 更强但新增路径；本轮不需要 |
| C | 仅改 taskAuth 登录跳转主机 / gateway→www 硬推 | 不能单独解决 authorize 无 Cookie；硬推已移除 |

## 实现要点

- `vite.config.js`：`VITE_TASK_GATEWAY_PUBLIC_BASE` ← `gatewayOrigin`；新增 `VITE_SSO_COOKIE_DOMAIN`
- `oidcResumeUrl.js` / `cookieUtils.js` / Django `auth_views`：共享 Domain + gateway 识别
- `Login.vue`：OIDC resume 前重写 `userId`
- `runAll/confload`（及 shareLib 同步）：`[]string` 模板解析
- `taskAuth`：`oidcLoginRedirectBase()` 只用已解析 www origins（跳过残留 `${`，无 gateway→www 兜底）
- E2E：`Gitlab.daydaymoney-cross-subdomain-sso.playwright.test.*`

## 架构影响

- 不新建 architecture target；属公网多子域 SSO 会话桥缺陷修复。
- 遵守 `.ai/01_project_constraints/17_cross_domain_frontend_api.md`。

## 验收

见同名 `.test-intent.md`、confload 单测与公网跨子域 Playwright。
