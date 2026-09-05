# Value Stream: taskAuth SSO 登录 404 修复 — 统一 OIDC 网关入口

> Derived from design: `docs/specs/oidc-sso-404-fix-design.md`
> **Status: ACTIVE** (2026-06-25)

## Value Summary

用户点击 GitLab「taskAuth SSO」按钮后，不再因 OIDC authorize 端点重定向到不存在的网关路径而看到 404 页面；OIDC 流量统一走 taskGateway（端口 18081），消除端口碎片化。

## Related Value Streams

- **`taskauth-oidc-issuer-docker-reachability`**: **modification** — 前置修复解决了 issuer Docker 可达性（`0.0.0.0` → `183.250.1.132:8003`）。本修复在此基础上将 issuer 统一到网关地址 `183.250.1.132:18081`，消除直连端口的碎片化。
- **`oidc-ssl-protocol-fix`**: **extension** — 前置修复解决了 SSL 协议问题（`SWD.url_builder = URI::HTTP`）。本修复增强 SSL fix 的持久化（容器重建后自动生效），并完成端到端 SSO 流程验证。
- **`task-gateway`**: **modification** — 网关新增 `/auth/login` 路由，确保 OIDC authorize 未认证重定向能正确转发到前端登录页。

## End-to-End Flow

```
[用户浏览器] → [主站登录 :4000] → [点击代码仓库]
  → [GitLab :8012/users/sign_in] → [点击 taskAuth SSO]
  → [GitLab OIDC discovery — GET :18081/.well-known/openid-configuration ✅]
  → [302 → :18081/api/oidc/authorize?...]
  → [taskAuth 认证检查]
    ├─ 已认证: 生成 code → 302 → GitLab callback → Dashboard ✅
    └─ 未认证: 302 → :18081/auth/login/?next=... → 前端登录页 ✅ (NO MORE 404)
```

### Before (broken):
```
未认证 → 302 → :18081/login/?next=...
  → 网关无 /login/ 路由 → fallthrough → Django → 404 ❌
```

### After (fixed):
```
未认证 → 302 → :18081/auth/login/?next=...
  → 网关路由 /auth/login → 前端登录页 → 用户登录 → redirect back ✅
```

## Value Stage Classification

- **Core value** — SSO 按钮点击后不再 404，已登录用户直接完成 SSO
- **Essential support** — 未登录用户正确重定向到登录页（而非 404）
- **Essential support** — OIDC issuer 走网关统一入口（安全 + 可观测性）
- **Essential support** — SSL fix 容器重建后自动恢复（持久化）
- **Enhancement** — Playwright E2E 测试覆盖完整 SSO 流程
- **Future** — 无需

## Value Increments

### Increment 1: 修复 OIDC Authorize 重定向 404 (Thin Slice)

**Value to user:** 点击「taskAuth SSO」不再看到 404 页面；未登录用户被正确引导到登录页。

**Scope:**
1. `taskAuth/src/oidc_handlers.go` — `loginURL` 从 `/login/` 改为 `/auth/login/`
2. `taskGateway/routes/routes.yaml` — 新增 `/auth/login` 路由（转发到前端或 Django）
3. Playwright 诊断测试 — 验证 SSO 点击后不出现 404

**Depends on:** `taskauth-oidc-issuer-docker-reachability` Increment 1 (issuer 已可达)

### Increment 2: 统一 OIDC Issuer 为网关地址

**Value to user:** OIDC 流量统一走网关（端口 18081），浏览器不再需要直连 taskAuth 端口 8003。

**Scope:**
1. `conf/auth/task-auth/config.yaml` — `oidc.issuer` 从 `http://183.250.1.132:8003` 改为 `http://183.250.1.132:18081`
2. `gitService/run.sh` — `GITLAB_OIDC_ISSUER` 同步更新（从 gateway publicBase 推导）
3. 回归验证 — 确保 GitLab 容器内通过网关也能完成 OIDC discovery

**Depends on:** Increment 1

### Increment 3: SSL Fix 持久化 + Playwright E2E

**Value to user:** GitLab 容器重建后 OIDC SSO 依然可用（无需手动重新注入 SSL fix）。

**Scope:**
1. GitLab entrypoint 注入 — 容器启动时自动应用 `SWD.url_builder = URI::HTTP`
2. Playwright E2E 测试 — 完整流程：主站登录 → 代码仓库 → taskAuth SSO → GitLab Dashboard
3. Playwright 404 回归断言 — 确保任何中间页面不出现 404

**Depends on:** Increment 2

## Impacted Existing Streams

| Stream | Impact |
|--------|--------|
| `taskauth-oidc-issuer-docker-reachability` | issuer URL 从 `:8003` 改为 `:18081`；`GITLAB_OIDC_ISSUER` 同步更新 |
| `oidc-ssl-protocol-fix` | SSL fix 持久化增强；Playwright 测试覆盖完整 SSO 流程（含 404 检测） |
| `task-gateway` > `gateway-routes-codegen` | 新增 `/auth/login` 路由；`apisix.yaml` 重新生成 |

## Field Changes

| Field | Change |
|-------|--------|
| `task-auth.runtime.oidc_issuer` | 值从 `http://183.250.1.132:8003` → `http://183.250.1.132:18081` |
| `git-service.runtime.oidc_issuer_env` | GITLAB_OIDC_ISSUER 默认值随 issuer 变化（run.sh 动态计算） |
| `task-gateway.routes.auth_login_page` | **新增**: `/auth/login` 路由，GET，auth_mode: none |
| `git-service.runtime.oidc_ssl_fix_persistent` | **新增**: SSL fix 容器 entrypoint 自动注入（不再依赖手动执行脚本） |
| `git-service.runtime.oidc_playwright_e2e_pass` | **扩展**: 增加 404 检测断言；完整 SSO → Dashboard 验证 |
