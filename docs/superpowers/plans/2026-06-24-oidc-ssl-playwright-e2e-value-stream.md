# Value Stream: OIDC SSL Protocol Fix — Playwright 验证 + E2E 测试

> Derived from design: `docs/specs/oidc-ssl-playwright-e2e-design.md`

## Value Summary

开发者和 CI 可以通过 Playwright 浏览器自动化验证 taskAuth SSO 登录 GitLab 的完整 OIDC 流程，并在任何部署环境中一键诊断和修复 `openid_connect` gem 的 HTTP→HTTPS 协议升级 bug。

## Related Value Streams

- **`oidc-ssl-protocol-fix`** (`2026-06-24-oidc-ssl-protocol-fix-value-stream.md`): **extension** — 原 value stream 定义了核心修复 (SWD.url_builder = URI::HTTP)，但状态仍为 `planned` 且缺少 Playwright 验证层。本 stream 在其基础上追加: (1) Playwright 诊断测试验证 OIDC discovery 可达性，(2) Playwright E2E 测试覆盖完整 SSO 登录流程，(3) 修复脚本持久化集成到 run.sh。原 Increment 2 (Initializer 持久化) 被本 stream 的实际实施所覆盖。
- **`taskauth-oidc-issuer-docker-reachability`**: **前置依赖** — issuer URL 可达性是本修复的前置条件。该 stream 已 active。

## End-to-End Flow

```
[开发者/CI 触发] → [Playwright 启动浏览器]
  → [导航到登录页 http://183.250.1.132:4000/auth/login/]
  → [填写凭证 + 勾选协议 + 点击登录]
  → [点击「代码仓库」→ 弹出 GitLab /users/sign_in]
  → [点击 taskAuth SSO 按钮]
  → [GitLab OIDC Discovery → fix 已生效，HTTP 协议正确]
  → [taskAuth 返回 discovery 文档 → 授权页面]
  → [回调 GitLab → 登录成功 ✅]
  → [Playwright 断言: 页面无 "record layer failure" 错误]
```

## Value Stage Classification

- **Core value** — Playwright E2E 测试覆盖完整 SSO 流程，CI 可自动回归
- **Essential support** — `fix_oidc_ssl.sh` 脚本: 幂等注入 initializer + reconfigure
- **Essential support** — Playwright 诊断测试: 快速定位 OIDC discovery 可达性
- **Enhancement** — run.sh 集成: 容器启动时自动执行修复
- **Future** — Gateway TLS 后 issuer 切换为 `https://`

## Value Increments

### Increment 1: Core Fix + Diagnostic Verification (Thin Slice)

**Value to user:** 修复可一键应用并验证 — `fix_oidc_ssl.sh` 注入 initializer，Playwright 诊断测试确认 OIDC discovery 使用 HTTP 协议。

**Scope:**
1. 创建 `gitService/scripts/fix_oidc_ssl.sh` — 注入 `zzz_fix_oidc_http.rb` + 幂等 reconfigure
2. 创建 `gitService/playwright/playwright.config.js` — Playwright 配置
3. 创建 `gitService/playwright/tests/oidc-ssl-diagnostic.playwright.test.js` — 诊断: discovery 可达 + SWD.url_builder 验证

**Depends on:** `taskauth-oidc-issuer-docker-reachability` (issuer 可达)

**Test verification:**
```bash
# 诊断测试
cd gitService/playwright && npx playwright test oidc-ssl-diagnostic
```

### Increment 2: E2E SSO Login Flow

**Value to user:** CI 可自动验证完整 SSO 登录流程 — 从登录页到 GitLab Dashboard，确保无 SSL 错误。

**Scope:**
1. 创建 `gitService/playwright/tests/oidc-sso-login.playwright.test.js` — 完整 SSO 登录 E2E
2. 创建 `gitService/playwright/tests/oidc-ssl-fix-verify.playwright.test.js` — 核心断言: 无 "record layer failure"

**Depends on:** Increment 1 (fix must be applied first)

**Test verification:**
```bash
cd gitService/playwright && npx playwright test oidc-ssl-fix-verify
```

### Increment 3: run.sh Integration (Persistence)

**Value to user:** GitLab 容器重建后自动恢复 OIDC protocol fix，无需手动干预。

**Scope:**
1. 修改 `gitService/run.sh` — GitLab 启动后自动调用 `fix_oidc_ssl.sh`
2. 更新 `conf/value-stream.yaml` — 标记 `oidc-ssl-protocol-fix` 为 `active`

**Depends on:** Increment 1

## Impacted Existing Streams

| Stream | Impact |
|--------|--------|
| `oidc-ssl-protocol-fix` | **activated** — 从 `planned` 变为 `active`，追加 Playwright test_file |
| `taskauth-oidc-issuer-docker-reachability` | 无变更 — 前置依赖，已 active |
| `gitlab-oauth-scope-failfast-governance` | **unblocked** — OIDC SSO 修复是其前置条件 |

## Field Changes

| Field | Change |
|-------|--------|
| `git-service.runtime.oidc_protocol_fix` | 状态从 planned → active; initializer 注入验证 |
| `git-service.runtime.oidc_ssl_initializer_present` | **新增** — `zzz_fix_oidc_http.rb` 存在性检查 |
| `git-service.runtime.oidc_playwright_e2e_pass` | **新增** — Playwright E2E 测试通过状态 |
