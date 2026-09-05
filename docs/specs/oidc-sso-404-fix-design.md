# taskAuth SSO 登录 404 修复 — 头脑风暴设计文档

**日期:** 2026-06-25
**状态:** 等待审批

---

## 1. 问题描述

用户从主站 `http://183.250.1.132:4000/auth/login/` 登录后，点击「代码仓库」进入 GitLab (`http://183.250.1.132:8012/users/sign_in`)，再点击「taskAuth SSO」登录按钮，浏览器跳转到新页面后提示 **"404 page not found"**。

### 复现步骤：
1. 浏览器访问 `http://183.250.1.132:4000/auth/login/`，输入账号密码登录
2. 登录成功后点击「代码仓库」
3. 弹出 GitLab 登录页 `http://183.250.1.132:8012/users/sign_in`
4. 点击「taskAuth SSO」按钮
5. 浏览器被重定向 → 显示 "404 page not found"

---

## 2. 架构分析

### 2.1 OIDC SSO 流程（当前）

```
Browser                  GitLab (:8012)          taskAuth (:8003)       Gateway (:18081)      Frontend (:4000)
  │                           │                       │                     │                    │
  │ ① GET /users/sign_in     │                       │                     │                    │
  │──────────────────────────>│                       │                     │                    │
  │                           │                       │                     │                    │
  │ ② 点击 "taskAuth SSO"    │                       │                     │                    │
  │──GET /users/auth/        │                       │                     │                    │
  │   openid_connect ───────>│                       │                     │                    │
  │                           │ ③ OIDC Discovery      │                     │                    │
  │                           │──GET /.well-known/    │                     │                    │
  │                           │   openid-config───>│                     │                    │
  │                           │<──200 JSON ──────────│                     │                    │
  │                           │                       │                     │                    │
  │ ④ 302 → issuer/api/oidc/ │                       │                     │                    │
  │   authorize?...           │                       │                     │                    │
  │<──────────────────────────│                       │                     │                    │
  │                           │                       │                     │                    │
  │ ⑤ GET :8003/api/oidc/    │                       │                     │                    │
  │   authorize?...           │                       │                     │                    │
  │──────────────────────────────────────────────────>│                     │                    │
  │                           │                       │                     │                    │
  │                    [认证检查]                      │                     │                    │
  │   ┌─ 已登录: 生成 code, 302 → GitLab callback     │                     │                    │
  │   └─ 未登录: 302 → Gateway /login/?next=... ──────────────────────────────>│                    │
  │                           │                       │                     │                    │
  │                           │                       │      ❌ /login/ 路由不存在 → 404         │
```

### 2.2 关键问题定位

**问题 #1（根因）: OIDC authorize 未认证重定向 URL 指向不存在的网关路径**

`taskAuth/src/oidc_handlers.go:171`：
```go
loginURL := cfg.GatewayPublicBase + "/login/?next=" + url.QueryEscape(r.URL.String())
http.Redirect(w, r, loginURL, http.StatusFound)
```

- `cfg.GatewayPublicBase` = `http://183.250.1.132:18081`（task-gateway 配置）
- 重定向到 `http://183.250.1.132:18081/login/?next=...`
- **网关没有 `/login/` 路由** → 请求 fallthrough 到 `django-default` (priority 0, `/*` → Django)
- Django 没有 `/login/` 页面 → 返回 **404 page not found**

**问题 #2（次要）: `OidcIssuer` 端口 8003 可能不对外暴露**

- taskAuth issuer 配置为 `http://183.250.1.132:8003`
- GitLab 的 OIDC discovery 返回 `authorization_endpoint: http://183.250.1.132:8003/api/oidc/authorize`
- 浏览器被重定向到 **端口 8003**，而非网关端口 18081
- 如果防火墙未开放 8003 → 连接失败；如果开放 → 能到达 taskAuth 但走的是直连而非网关路由
- 对比：网关在 18081 已配置完整的 OIDC 路由（oidc-authorize, oidc-token, oidc-userinfo, oidc-jwks, oidc-discovery）

**问题 #3（预存但可能已修复）: OIDC SSL protocol fix 可能未持久化**

- `fix_oidc_ssl.sh` 注入 `SWD.url_builder = URI::HTTP` 到 GitLab Rails initializer
- GitLab 容器重建后（`docker compose down && up`），注入的 init 文件丢失
- 若 SSL fix 失效 → GitLab OIDC discovery 尝试 HTTPS → SSL record layer failure → 间接导致 404

---

## 3. 修复方案

### 方案 A（推荐）: 统一 OIDC 端点走网关 + 修正登录重定向

**修改 #1: taskAuth OIDC Issuer 改为网关地址**

`conf/auth/task-auth/config.yaml`:
```yaml
oidc:
  issuer: "http://183.250.1.132:18081"  # 改为网关 publicBase，而非 taskAuth 直连端口
```

**修改 #2: OIDC authorize 登录重定向改为前端登录页**

`taskAuth/src/oidc_handlers.go:171`:
```go
// Before:
loginURL := cfg.GatewayPublicBase + "/login/?next=" + url.QueryEscape(r.URL.String())

// After:
loginURL := cfg.GatewayPublicBase + "/auth/login/?next=" + url.QueryEscape(r.URL.String())
```

同时需要在 taskGateway 的 `routes.yaml` 添加 `/auth/login/` 路由（或确保 catch-all 能正确转发到前端）。

**修改 #3: 网关增加 login 页面路由（可选回退）**

`taskGateway/routes/routes.yaml` 增加：
```yaml
- id: frontend-login-page
  priority: 800
  uri: /auth/login
  methods: [GET]
  upstream: taskFE  # 或 redirect 到 4000 端口
  auth_mode: none
```

**修改 #4: 确保 OIDC SSL fix 持久化**

将 `SWD.url_builder = URI::HTTP` 注入逻辑集成到 GitLab docker-compose 初始化脚本或 entrypoint，使其在容器重建后自动生效。

### 方案 B（最小改动）: 仅修正登录重定向 URL

只修改 `handleOidcAuthorize` 中的登录重定向 URL，改为直接跳转到前端登录页：

```go
loginURL := "http://183.250.1.132:4000/auth/login/?next=" + url.QueryEscape(r.URL.String())
```

**缺点**: 硬编码 URL，不灵活；且 issuer 端口问题依然存在。

### 方案对比

| 维度 | 方案 A | 方案 B |
|------|--------|--------|
| 修复根因 | ✅ 修复 issuer + 登录重定向 | ⚠️ 仅修复登录重定向 |
| 端口暴露 | ✅ 统一走网关 18081 | ❌ 仍需开放 8003 |
| 可配置性 | ✅ 从配置读取 | ❌ 硬编码 |
| SSL fix 持久化 | ✅ 一并处理 | ❌ 不涉及 |
| 工作量 | 中等（4 处改动） | 小（1 处改动） |
| 回归风险 | 低 | 极低 |

**推荐方案 A**，因为它在架构层面统一了 OIDC 流量的入口（都走网关），消除了端口碎片化问题。

---

## 4. Domain Concept Inventory（领域概念清单）

| 类别 | 概念 |
|------|------|
| **Bounded Context** | `auth` — OIDC Provider (taskAuth), `git-integration` — GitLab OmniAuth Consumer |
| **Key Entities** | `OidcClient` (client_id, redirect_uris), `AuthorizationCode`, `User` |
| **Candidate Aggregates** | `OidcClient` 聚合根（管理 redirect_uris、client_secret）|
| **Domain Events** | `UserAuthenticatedViaOIDC`（GitLab 通过 OIDC 成功认证用户）|

---

## 5. Value Stream 影响分析

### 受影响的价值流

| Value Stream | 影响 |
|-------------|------|
| `taskauth-oidc-issuer-docker-reachability` | **直接修改**: OIDC issuer 从 `:8003` 改为 gateway `:18081` |
| `oidc-ssl-protocol-fix` | **关联**: SSL fix 持久化增强，增加容器重建后自动恢复机制 |
| `task-gateway` | **新增路由**: `/auth/login` 页面路由 |

### 字段影响

| 字段 | 变更 |
|------|------|
| `task-auth.runtime.oidc_issuer` | 值从 `http://183.250.1.132:8003` → `http://183.250.1.132:18081` |
| `git-service.runtime.oidc_issuer_env` | GITLAB_OIDC_ISSUER 需同步更新（run.sh 已动态计算，无需手动改） |
| `task-gateway.routes.login_page` | 新增 `/auth/login` 路由项 |

### 测试影响

| 测试文件 | 操作 |
|----------|------|
| `gitService/playwright/tests/oidc-sso-login.playwright.test.js` | **更新**: 增加 404 检测断言 + 完整 SSO 成功流程验证 |
| `gitService/playwright/tests/oidc-ssl-fix-verify.playwright.test.js` | **更新**: 验证 issuer URL 走网关 |
| **新增** `gitService/playwright/tests/oidc-sso-e2e-full-flow.playwright.test.js` | **新建**: 端到端 SSO 测试（登录 → 代码仓库 → SSO → GitLab Dashboard） |

---

## 6. 实施计划概要

1. **修改 taskAuth 配置**: issuer 改为 gateway publicBase
2. **修改 taskAuth 代码**: OIDC authorize 登录重定向改为 `/auth/login/`
3. **修改 taskGateway 路由**: 新增 `/auth/login` 页面路由（转发到前端）
4. **SSL fix 持久化**: 将 `SWD.url_builder` 初始化注入到 docker-compose entrypoint
5. **Playwright 诊断测试**: 模拟完整 SSO 流程，验证不再出现 404
6. **Playwright 端到端测试**: 完整流程（登录 → 代码仓库 → SSO → GitLab Dashboard）

---

## 7. 验收标准

- [ ] 用户点击「taskAuth SSO」后不再出现 404
- [ ] 已登录用户 SSO 流程顺利完成，最终到达 GitLab Dashboard
- [ ] 未登录用户被正确重定向到登录页（而非 404）
- [ ] Playwright E2E 测试全部通过
- [ ] OIDC discovery 端点通过网关可达
- [ ] GitLab 容器重建后 SSL fix 自动生效
