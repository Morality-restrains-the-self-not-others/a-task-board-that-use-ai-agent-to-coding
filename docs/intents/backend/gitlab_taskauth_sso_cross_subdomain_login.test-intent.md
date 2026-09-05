# 测试意图：GitLab taskAuth SSO 跨子域重复登录

## 对应功能意图

`gitlab_taskauth_sso_cross_subdomain_login.intent.md`

## 用例

### T1 — 前端 sanitize 接受生产 api authorize URL

- **给定** `VITE_TASK_GATEWAY_PUBLIC_BASE=https://api.daydaymoney.com`
- **当** `sanitizeOidcResumeNext('https://api.daydaymoney.com/api/oidc/authorize?...')`
- **则** 返回原 URL（非 null）

### T2 — 空构建 env 时从 www 推导 api gateway

- **给定** gateway/api env 为空，`window.location.hostname=www.daydaymoney.com`
- **当** sanitize / resolve
- **则** gateway base 为 `https://api.daydaymoney.com`

### T3 — Cookie Domain 派生

- **给定** hostname=`www.daydaymoney.com`
- **则** `resolveSharedCookieDomain` → `.example.com`
- **给定** IP hostname
- **则** 返回空（host-only）

### T4 — Django 已登录 OIDC resume 写共享 Domain

- **给定** `SSO_COOKIE_DOMAIN=.example.com` 且已登录
- **当** `GET /auth/login/?next=https://api.daydaymoney.com/api/oidc/authorize...`
- **则** 302 到 authorize，且 `userId` cookie domain 为 `.example.com`

### T5 — taskAuth 未登录优先 www 登录入口

- **给定** 已解析的 `PostLogoutOrigins` 含 `https://www.daydaymoney.com`
- **当** 无凭证访问 `/api/oidc/authorize`
- **则** Location 以 `https://www.daydaymoney.com/auth/login/?next=` 开头

### T6 — confload 解析 `[]string` 模板

- **给定** `postLogoutRedirectOrigins: ["${scheme}://${subdomains.www}", ...]`
- **当** `ReadAppConfigResolved`
- **则** 得到 `https://www.daydaymoney.com` 等已展开值（不得残留 `${`）

### T7 — 公网 www+api Playwright

- **给定** `E2E_SITE_ORIGIN=https://www.daydaymoney.com`，`GATEWAY_URL=https://api.daydaymoney.com`
- **当** 登录后查 `context.cookies(gateway)`
- **则** 存在 `userId` 且 Domain 为 `.example.com`；直开 authorize 不落回登录页；项目列表→代码仓库→SSO 进入 GitLab

## 自动化落点

- `task2app/front_project/app/src/utils/ssoCookieDomain.test.js`
- `task2app/front_project/app/src/utils/oidcResumeUrl.test.js`
- `task2app/Saas_project/tests/test_auth_login_oidc_resume.py`
- `taskAuth/src/oidc_provider_test.go`
- `runAll/confload/load_test.go`（`TestReadAppConfigResolvedStringSlice`）
- `task2app/playwright/front_project/tests/Gitlab.daydaymoney-cross-subdomain-sso.playwright.test.js`
- `task2app/playwright/front_project/tests/Gitlab.daydaymoney-cross-subdomain-sso.playwright.test.sh`

## 变更记录

- 2026-07-12：初版。
- 2026-07-12：补 T6 confload / T7 公网跨子域 E2E。
