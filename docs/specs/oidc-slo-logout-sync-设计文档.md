# OIDC Single Logout (SLO) 登出同步 — 设计文档

## 1. 问题描述

### 复现步骤

1. 用户访问 `http://183.250.1.132:4000/auth/login/`，使用账号 `contact@daydaymoney.com` 登录主应用
2. 点击导航栏「代码仓库」，跳转到 `http://183.250.1.132:8012/users/sign_in`
3. 点击「taskAuth SSO」按钮，通过 OIDC SSO 登录 GitLab
4. 登录成功后自动跳回 `http://183.250.1.132:4000/projects/`
5. 点击右上角「退出」按钮
6. **Bug**: 主应用退出后，刷新 `http://183.250.1.132:8012/`，GitLab **仍然保持登录状态**

### 预期行为

主应用退出后，通过 OIDC SSO 登录的 GitLab 也应一并退出（Single Logout）。

---

## 2. 根因分析

这是一个**三层 SLO 全部缺失**的问题：

```
┌─────────────────────────────────────────────────────┐
│  Layer 1: 主应用 (task2app Django :4000)              │
│  ├─ /logout/ view 是空函数体（只有 import，无逻辑）      │
│  ├─ Django session 从未在服务端销毁                     │
│  ├─ userId cookie 仅在客户端清除（JS delete cookie）     │
│  └─ 登出后仅清 localStorage + 跳转 /auth/login/         │
├─────────────────────────────────────────────────────┤
│  Layer 2: taskAuth OIDC Provider (Go :18081)         │
│  ├─ handleLogout 仅删除 accounts_customtoken 记录      │
│  ├─ OIDC Discovery 无 end_session_endpoint            │
│  ├─ 无 RP-Initiated Logout                           │
│  ├─ 无 Back-Channel Logout                            │
│  ├─ 无 Front-Channel Logout                           │
│  ├─ ID Token 缺少 sid (session_id) claim              │
│  └─ JWT 无状态，无撤销机制                              │
├─────────────────────────────────────────────────────┤
│  Layer 3: GitLab (gitService :8012)                   │
│  ├─ OmniAuth OIDC 未配置 end_session_endpoint         │
│  ├─ 登出后 GitLab session 完全不受影响                  │
│  └─ GitLab 本身支持 RP-Initiated Logout，但未启用       │
└─────────────────────────────────────────────────────┘
```

### 核心文件

| 文件 | 问题 |
|------|------|
| `task2app/Saas_project/frontend_app/views/auth_views.py:251` | `logout()` 函数体为空 — 不销毁 Django session |
| `task2app/front_project/app/src/components/Navbar.logic.vue:211-280` | 前端登出仅清客户端状态 |
| `taskAuth/src/auth_login.go:133-147` | `handleLogout` 仅删本地 token，不通知 RP |
| `taskAuth/src/oidc_handlers.go:85-101` | OIDC Discovery 无 `end_session_endpoint` |
| `gitService/docker-compose.yml:39-66` | GitLab OmniAuth 无 `end_session_endpoint` 配置 |

---

## 3. 修复方案

### 总体策略：Redirect Chain Logout（重定向链式登出）

采用 OIDC **RP-Initiated Logout** 规范，通过浏览器重定向链依次登出三方：

```
用户点击「退出」
  │
  ▼
主应用 POST /api/accounts/users/logout/
  ├─ 转发至 taskAuth 销毁 token
  ├─ 服务端销毁 Django session (auth_logout)
  ├─ 清除 sessionid / userId cookie
  └─ 返回 { redirect_url: taskAuth end_session }
  │
  ▼
浏览器重定向至 taskAuth GET /api/oidc/endsession
  ├─ 清除 taskAuth userId cookie
  ├─ 销毁 accounts_customtoken
  └─ 302 重定向至 GitLab /users/sign_out
  │
  ▼
浏览器重定向至 GitLab GET /users/sign_out
  ├─ GitLab 销毁自身 session
  └─ 302 重定向回主应用 /auth/login/
  │
  ▼
用户回到登录页，三方全部登出
```

### 3.1 修改 taskAuth（Go）— 新增 OIDC EndSession 端点

**文件**: `taskAuth/src/oidc_handlers.go`

新增 `handleOidcEndSession` 处理函数：

```go
// GET /api/oidc/endsession
// 实现 OIDC RP-Initiated Logout
func handleOidcEndSession(w http.ResponseWriter, r *http.Request) {
    // 1. 解析参数
    postLogoutRedirectURI := r.URL.Query().Get("post_logout_redirect_uri")
    
    // 2. 清除 userId cookie（跨端口 SSO 桥）
    http.SetCookie(w, &http.Cookie{
        Name:     "userId",
        Value:    "",
        Path:     "/",
        MaxAge:   -1,
        HttpOnly: false,
    })
    
    // 3. 销毁 taskAuth token（从 Authorization header 或 token cookie 获取）
    key := extractTokenFromRequest(r)
    if key != "" {
        _ = deleteTokenByKey(key)
    }
    
    // 4. 构造 GitLab 登出 URL
    gitlabSignOutURL := cfg.GitServicePublicBase + "/users/sign_out"
    if postLogoutRedirectURI != "" {
        gitlabSignOutURL += "?redirect_uri=" + url.QueryEscape(postLogoutRedirectURI)
    }
    
    // 5. 302 重定向至 GitLab 登出
    http.Redirect(w, r, gitlabSignOutURL, http.StatusFound)
}
```

**文件**: `taskAuth/src/oidc_handlers.go:85-101`

在 OIDC Discovery 中添加 `end_session_endpoint`：

```go
doc := map[string]interface{}{
    // ...现有字段...
    "end_session_endpoint": cfg.Issuer + "/api/oidc/endsession",
}
```

**文件**: `taskAuth/src/handlers.go`

注册新路由：

```go
mux.HandleFunc("GET /api/oidc/endsession", handleOidcEndSession)
```

**文件**: `taskAuth/src/config.go`

添加配置字段：

```go
GitServicePublicBase string  // GitLab 外部 URL，如 http://183.250.1.132:8012
```

### 3.2 修改主应用（Django）— 实现真正的服务端登出

**文件**: `task2app/Saas_project/frontend_app/views/auth_views.py`

重写 `logout()` 视图：

```python
def logout(request):
    from django.conf import settings as dj_settings
    from django.contrib.auth import logout as auth_logout
    from django.utils.http import url_has_allowed_host_and_scheme
    
    # 1. 销毁 Django session
    auth_logout(request)
    
    # 2. 清除 cookies
    response = HttpResponseRedirect('/auth/login/')
    response.delete_cookie('userId')
    response.delete_cookie('sessionid')
    
    return response
```

**文件**: `task2app/Saas_project/accounts/views/user_views.py`

`UserViewSet.logout()` — 当前返回 501，改为转发至 taskAuth 并返回 redirect_url：

```python
@action(detail=False, methods=['post'])
def logout(self, request):
    """用户登出：转发至 taskAuth + 销毁 Django session + 返回 SLO 重定向 URL"""
    # 1. 转发登出到 taskAuth
    token = request.headers.get('Authorization', '').replace('Token ', '')
    if token:
        forward_to_taskauth('POST', '/api/accounts/users/logout/', 
                           headers={'Authorization': f'Token {token}'})
    
    # 2. 销毁 Django session
    auth_logout(request)
    
    # 3. 返回 GitLab sign_out URL 供前端重定向
    git_service_base = dj_settings.GIT_SERVICE_PUBLIC_BASE  # 新增配置
    sign_out_url = f"{git_service_base}/users/sign_out"
    
    return Response({
        'detail': 'ok',
        'slo_redirect_url': sign_out_url,
    })
```

### 3.3 修改前端（Vue）— 登出后重定向至 GitLab 登出

**文件**: `task2app/front_project/app/src/components/Navbar.logic.vue`

在 `handleLogout` 成功回调中，执行 SLO 重定向链：

```javascript
async function handleLogout() {
  const token = localStorage.getItem('authToken') || ''
  try {
    const resp = await axios.post('/api/accounts/users/logout/', null, {
      headers: token ? { Authorization: `Token ${token}` } : {}
    })
    // 成功后清理
    clearLocalAuthState()
    
    // SLO: 先跳 GitLab sign_out，GitLab 会 302 回 /auth/login/
    const sloUrl = resp.data?.slo_redirect_url
    if (sloUrl) {
      window.location.href = sloUrl + '?redirect_uri=' + encodeURIComponent(window.location.origin + '/auth/login/')
    } else {
      window.location.href = '/auth/login/'
    }
  } catch {
    clearLocalAuthState()
    window.location.href = '/auth/login/'
  }
}
```

### 3.4 修改 GitLab 配置（可选增强）

**文件**: `gitService/docker-compose.yml`

在 OmniAuth OIDC provider args 中添加：

```ruby
args: {
  # ...现有配置...
  client_options: {
    # ...现有配置...
    end_session_endpoint: ENV['GITLAB_OIDC_ISSUER'] + '/api/oidc/endsession',
  }
}
```

**注意**: 由于我们通过直接跳转 `/users/sign_out` 来登出 GitLab，此配置是可选的增强项。GitLab 的原生 RP-Initiated Logout 支持需要此配置，但我们的重定向链方案不依赖它。

---

## 4. Playwright 端到端测试

### 4.1 核验测试（Bug 复现）

**文件**: `gitService/playwright/tests/oidc-slo-logout-bug-repro.playwright.test.js`

验证当前退出主应用后 GitLab 仍保持登录（复现 bug，作为基线）。

### 4.2 E2E 测试（修复验证）

**文件**: `gitService/playwright/tests/oidc-slo-logout-sync.playwright.test.js`

完整登出同步 E2E 测试：

```
测试用例: Logout from main app should also logout from GitLab (SLO)

Steps:
  1. 打开主应用登录页 http://183.250.1.132:4000/auth/login/
  2. 输入邮箱和密码，点击登录
  3. 验证跳转至 /projects/
  4. 新标签页打开 http://183.250.1.132:8012
  5. 点击 taskAuth SSO 登录
  6. 验证 GitLab Dashboard 可见（已登录）
  7. 切回主应用标签页
  8. 点击导航栏「退出」
  9. 等待重定向链完成，回到登录页
  10. 刷新 GitLab 页面
  11. 断言: GitLab 显示登录页（已登出）

SLO 核心断言:
  - GitLab 页面 URL 不再是 /dashboard 或 root
  - GitLab 页面出现 "sign_in" 或 "登录" 文本
  - GitLab 页面的 session cookie 已清除
```

### 4.3 测试配置

测试将使用 Playwright 配置文件，从环境变量读取：
- `TEST_USER_EMAIL`: `contact@daydaymoney.com`
- `TEST_USER_PASSWORD`: `rgNodkdq8677!ci`
- `MAIN_APP_URL`: `http://183.250.1.132:4000`
- `GITLAB_URL`: `http://183.250.1.132:8012`

---

## 5. 域概念清单

### Bounded Contexts
- **认证上下文 (Auth Context)**: taskAuth OIDC Provider — 用户身份与令牌管理
- **主应用上下文 (Main App Context)**: task2app Django — 业务会话管理
- **代码仓库上下文 (Git Service Context)**: GitLab — 第三方 RP 会话

### Key Entities
- **User** (taskAuth): `accounts_user` 表，用户身份真源
- **CustomToken** (taskAuth): `accounts_customtoken` 表，API 认证令牌（单例）
- **OIDC Client** (taskAuth): `oidc_client` 表，OIDC 依赖方注册
- **OIDC Authorization** (taskAuth): `oidc_authorization` 表，授权码（一次性）

### Candidate Aggregates
- **UserSession** (跨上下文): 用户会话聚合 — 跨三个系统的登出一致性边界
  - Root: taskAuth CustomToken
  - 关联: Django Session (主应用) + GitLab Session (RP)

### Domain Events
- **UserLoggedOut**: taskAuth 发出，通知所有 RP 登出
  - 替代方案：Browser Redirect Chain（不需要异步事件，用同步重定向）

---

## 6. 价值流影响分析

### 受影响的现有价值流

| Stream | 影响 |
|--------|------|
| `user-auth` (用户与认证) | 新增 `slo-logout` step — 单点登出端到端闭合 |
| `oidc-sso-404-fix` (平台与本地开发) | 关联 — 同属 OIDC SSO 体系完善 |
| `oidc-ssl-protocol-fix` (平台与本地开发) | 关联 — 同属 OIDC 基础设施 |

### 需要新增的价值流

建议在 `value-stream.yaml` 的「用户与认证」域或「平台与本地开发」域下新增：

```yaml
- name: oidc-slo-logout-sync
  domain: 用户与认证
  description: OIDC 单点登出同步 — 主应用登出时联动 GitLab 登出
  steps:
  - name: taskauth-end-session-endpoint
    status: planned
    test_file: taskAuth/src/oidc_handlers_test.go
    fields:
    - name: task-auth.runtime.end_session_endpoint
      description: OIDC Discovery 新增 end_session_endpoint
    - name: task-auth.accounts_customtoken.key
      description: EndSession 清除 token 记录
  - name: main-app-server-side-logout
    status: planned
    test_file: accounts/view_test/UserViewSet_logout_test.py
    fields:
    - name: saas-backend.runtime.logout_session_destroyed
      description: Django session 服务端销毁
    - name: task-auth.accounts_customtoken.key
      description: 登出时转发销毁 taskAuth token
  - name: gitlab-slo-config
    status: planned
    test_file: ../../gitService/scripts/test_sync_oauth_app.sh
    fields:
    - name: git-service.runtime.oidc_end_session_endpoint
      description: GitLab OmniAuth OIDC end_session_endpoint 配置
  - name: playwright-slo-e2e
    status: planned
    test_file: ../../gitService/playwright/tests/oidc-slo-logout-sync.playwright.test.js
    fields:
    - name: git-service.runtime.oidc_slo_playwright_e2e_pass
      description: Playwright E2E 验证主应用登出后 GitLab 同步登出
```

### 字段影响

| 服务.表.字段 | 变更 |
|-------------|------|
| `task-auth.accounts_customtoken.key` | EndSession 时读 + 删 |
| `task-auth.runtime.end_session_endpoint` | 新增运行时配置项 |
| `saas-backend.runtime.git_service_public_base` | 新增配置（GitLab 外部 URL） |
| `git-service.runtime.oidc_end_session_endpoint` | 新增 OmniAuth 配置项 |

### 测试影响

| 文件 | 类型 | 说明 |
|------|------|------|
| `gitService/playwright/tests/oidc-slo-logout-bug-repro.playwright.test.js` | 新增 | Bug 复现测试 |
| `gitService/playwright/tests/oidc-slo-logout-sync.playwright.test.js` | 新增 | SLO 修复 E2E 验证 |
| `accounts/view_test/UserViewSet_logout_test.py` | 可能需要更新 | logout API 现在返回 redirect_url |
| `taskAuth/src/oidc_handlers_test.go` | 新增 | EndSession handler 单元测试 |

---

## 7. 实施优先级

| 优先级 | 修改点 | 理由 |
|--------|--------|------|
| P0 | taskAuth: 新增 `handleOidcEndSession` | SLO 核心 — 没有此端点无法联动登出 |
| P0 | taskAuth: OIDC Discovery 添加 `end_session_endpoint` | 标准合规 + 方便 GitLab 自动发现 |
| P0 | 主应用 Django: `logout()` 视图实现真正的 session 销毁 | 当前空函数体是最直接的 bug |
| P1 | 前端: 登出后重定向至 GitLab sign_out | 完成端到端登出链 |
| P1 | Playwright E2E 测试 | 回归保护 |
| P2 | GitLab OmniAuth 配置 `end_session_endpoint` | 可选增强 |

---

## 8. 风险与注意事项

1. **GitLab `/users/sign_out` 的 redirect_uri 安全性**: GitLab 可能限制 `redirect_uri` 只能是白名单域名。需验证或改用直接跳转。
2. **重定向链的用户体验**: 用户会看到短暂的中间页面跳转（taskAuth → GitLab → 主应用）。可通过减少中间跳转来优化。
3. **JWT 令牌撤销**: 当前 taskAuth 的 JWT access token 无法撤销（无状态），在 TTL 内理论上仍可用。但 SLO 更关注的是浏览器会话层面的登出，而非 token 层面的撤销。完整的 token 撤销需引入黑名单机制（工作量较大，本次不做）。
4. **Cookie 清除的域问题**: `userId` cookie 在相同根域下共享，如果主应用和 taskAuth 在不同子域可能需要调整 path/domain。

---

## 9. 设计决策记录

| 决策 | 选择 | 替代方案 | 理由 |
|------|------|---------|------|
| SLO 实现方式 | Redirect Chain (浏览器重定向) | Back-Channel Logout (服务器间通知) | Redirect Chain 简单直接，不需要 token 撤销基础设施；Back-Channel 需要 taskAuth 维护 RP 列表并逐一下发 |
| GitLab 登出方式 | 直接跳转 `/users/sign_out` | 配置 `end_session_endpoint` 走标准 RP-Initiated Logout | 直接跳转最简单可靠；GitLab 的 `end_session_endpoint` 支持作为可选增强 |
| 是否实现 token 黑名单 | 否 | 是 (用 Redis 存黑名单) | 本次聚焦浏览器会话层 SLO；token 撤销是独立需求 |
| 是否使用 OIDC Session Management | 否 | 是 (RP iframe + OP iframe) | 复杂度高，浏览器支持差；Redirect Chain 方案更直观 |
