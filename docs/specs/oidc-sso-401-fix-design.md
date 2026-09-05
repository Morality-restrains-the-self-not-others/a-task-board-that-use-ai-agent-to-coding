# Design Document: OIDC SSO 登录 HTTP ERROR 401 修复

**Created:** 2026-06-25  
**Status:** 待审批  
**Author:** Claude (from user ljy)

---

## 1. 问题描述

用户通过以下流程触发错误：

1. 浏览器访问 `http://183.250.1.132:4000/auth/login/` → 登录成功
2. 点击「代码仓库」→ 打开 GitLab 登录页 `http://183.250.1.132:8012/users/sign_in`
3. 点击「taskAuth SSO」按钮
4. GitLab 302 跳转到 `http://183.250.1.132:18081/api/oidc/authorize?...`
5. taskAuth OIDC handler 发现用户未认证 → 302 跳转到 `/auth/login/?next=...`
6. 浏览器请求 `http://183.250.1.132:18081/auth/login/?next=...` → **返回 HTTP ERROR 401**

### Playwright 复现结果

```
Step 1-4: 登录 183.250.1.132:4000 ✅ 成功
Step 5:   点击「代码仓库」→ GitLab sign_in ✅ 成功
Step 6:   点击「taskAuth SSO」→ OIDC authorize 跳转 ✅ 
          302: /users/auth/openid_connect → /api/oidc/authorize?...
          302: /api/oidc/authorize?... → /auth/login/?next=...
          401: /auth/login/?next=...                             🔴 401!
```

---

## 2. 根因分析

### 2.1 核心原因：APISIX 路由 `uri` 精确匹配 vs 重定向 URL 尾部斜杠

**taskGateway routes 配置** (`routes/routes.yaml:183-188`):
```yaml
- id: auth-login-page
  priority: 800
  uri: /auth/login          # ← 精确匹配，不含尾部斜杠
  methods: [GET]
  upstream: django
  auth_mode: none           # 公开页面，无需认证
```

**taskAuth OIDC handler 重定向代码** (`oidc_handlers.go:171`):
```go
loginURL := cfg.GatewayPublicBase + "/auth/login/?next=" + url.QueryEscape(r.URL.String())
// 生成: http://183.250.1.132:18081/auth/login/?next=...
//                                            ↑ 路径是 /auth/login/
```

### 2.2 路由匹配失败流程

```
浏览器请求: GET /auth/login/?next=...
                     ↓
APISIX 路由匹配:
  ├── /auth/login (exact match) → ❌ 不匹配 /auth/login/
  ├── /* (catch-all, priority: 0) → ✅ 匹配
  └── django-default: auth_mode=token → forward-auth 拦截 → 401
```

APISIX 中 `uri: /auth/login` 是**精确匹配**，只匹配路径 `/auth/login`，不匹配 `/auth/login/`。因此带尾部斜杠的请求落入 catch-all 路由 `django-default`（`/*`），该路由配置了 `auth_mode: token`，触发 forward-auth 校验，但请求中没有有效的 `Authorization: Token xxx` header，返回 401。

### 2.3 产生 `Authorization` header 失败的原因

APISIX forward-auth 只透传 `Authorization` header 给 taskAuth。302 重定向是浏览器发起的普通 GET 请求：
- 浏览器不会在重定向跟随中携带自定义 `Authorization` header
- 浏览器会携带同域 Cookie，但 APISIX 没有配置从 Cookie 提取 token 注入 `Authorization` header 的逻辑
- 跨端口（4000 → 18081）的 session 隔离加剧了此问题

### 2.4 历史背景

这是 OIDC SSO 系列的第三个问题（详见记忆文件）：
1. **SSL Record Layer Failure** → 修复：GitLab 容器内 `SWD.url_builder = URI::HTTP`
2. **404 Page Not Found** → 修复：重定向从 `/login/` 改为 `/auth/login/` + 添加路由
3. **401 Unauthorized**（当前）→ 修复：路由尾部斜杠匹配

issue #2 修复时添加了 `/auth/login` 路由，但 URL 模板用了 `/auth/login/`（带斜杠），造成本次问题。

---

## 3. 修复方案

### 3.1 路由修复（主修复）

**文件**: `taskGateway/routes/routes.yaml`

将 `auth-login-page` 从单一路由改为同时支持两种形式：

```yaml
# Before
- id: auth-login-page
  priority: 800
  uri: /auth/login
  methods: [GET]
  upstream: django
  auth_mode: none

# After
- id: auth-login-page
  priority: 800
  uris:                          # uris (复数) 替代 uri
    - /auth/login
    - /auth/login/
  methods: [GET]
  upstream: django
  auth_mode: none
```

### 3.2 重定向 URL 硬化（防御性修复）

**文件**: `taskAuth/src/oidc_handlers.go`

移除尾部斜杠，与路由配置保持一致：

```go
// Before (line 171)
loginURL := cfg.GatewayPublicBase + "/auth/login/?next=" + url.QueryEscape(r.URL.String())

// After
loginURL := cfg.GatewayPublicBase + "/auth/login?next=" + url.QueryEscape(r.URL.String())
```

### 3.3 部署步骤

1. 修改 `routes/routes.yaml`（路由配置）
2. 运行 `python3 scripts/routes-to-apisix.py` 重新生成 `apisix.yaml`
3. 重新编译 taskAuth: `cd taskAuth && go build -o taskAuth ./src/`
4. 重启服务: `runAll` 或手动重启 taskGateway + taskAuth
5. Playwright 端到端测试验证

---

## 4. Playwright 端到端测试

### 4.1 测试文件

**新增**: `gitService/playwright/tests/oidc-sso-401-fix-verify.playwright.test.js`

测试内容：
- **TC-01**: 完整 SSO 登录流程（login → 代码仓库 → taskAuth SSO → GitLab Dashboard）
- **TC-02**: 验证无 401 错误（检查 status code 和 body text）
- **TC-03**: 验证无 404 回归（确保之前的 404 fix 未被破坏）
- **TC-04**: 验证无 SSL 错误回归
- **TC-05**: 验证 `/auth/login` 和 `/auth/login/` 均返回 200
- **TC-06**: 验证 `/auth/login/?next=<url>` 正常返回登录页（非 401）

### 4.2 测试环境变量

- `LOGIN_URL=http://183.250.1.132:4000/auth/login/`
- `PW_EMAIL=contact@daydaymoney.com`
- `PW_PASSWORD=rgNodkdq8677!ci`
- `GITLAB_URL=http://183.250.1.132:8012`
- `GATEWAY_URL=http://183.250.1.132:18081`

---

## 5. 影响范围

### 5.1 修改文件清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `taskGateway/routes/routes.yaml` | 修改 | auth-login-page 路由支持带/不带斜杠 |
| `taskGateway/apisix/apisix.yaml` | 重新生成 | routes-to-apisix.py 生成 |
| `taskAuth/src/oidc_handlers.go` | 修改 | 移除重定向 URL 尾部斜杠 |
| `gitService/playwright/tests/oidc-sso-401-fix-verify.playwright.test.js` | 新增 | E2E 验证测试 |

### 5.2 未受影响

- GitLab 容器内 OIDC 配置（不需要变更）
- taskAuth OIDC bootstrap client 配置
- taskGateway 其他路由配置
- Django/Vue 前端代码

---

## 6. 域概念清单（供 /5-ddd 使用）

- **Bounded Contexts**: Auth (taskAuth), Gateway (taskGateway), Git Service (gitService/GitLab)
- **Key Entities**: User (taskAuth), OidcClient (taskAuth - gitlab-git-service)
- **Candidate Aggregates**: GatewayRoute → routes config; OidcAuthorization → code + client + user
- **Domain Events**: OidcAuthorizationRequested, UserAuthenticatedForSSO

---

## 7. Value Stream 影响分析

**影响的 streams**: `user-auth` stream (step: oidc-sso-login)

| 维度 | 评估 |
|------|------|
| 哪些 stream 受影响 | `user-auth` — OIDC SSO 登录步骤 |
| 是否需要新 stream | 否 |
| Fields 影响 | 无新字段 |
| 测试影响 | 新增 `oidc-sso-401-fix-verify.playwright.test.js` |
| 状态变更 | 将 oidc-sso-login step 从 planned→active（如果还是 planned） |
| 跨 stream 依赖 | 否 |

---

## 8. 审批

请审批以上设计，审批后将进入实现阶段。
