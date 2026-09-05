# 实施计划: OIDC SLO 登出同步

> 输入:
> - 设计文档: `docs/specs/oidc-slo-logout-sync-设计文档.md`
> - 价值流: `docs/superpowers/plans/2026-06-25-oidc-slo-logout-sync-value-stream.md`
> - 领域模型: `docs/superpowers/plans/2026-06-25-oidc-slo-logout-sync-ddd-model.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-25-oidc-slo-logout-sync-nfr-clarification.md`

## 前置检查

- [x] Domain 层: `taskAuth/domain/session_termination.go` + tests (PASS)
- [x] Domain 层: `taskAuth/domain/customtoken_repository.go` (repository interface)
- [ ] 所有后续任务待完成

---

## Increment 1: taskAuth EndSession 端点 (Thin Slice)

### Task 1.1: 添加 GitServicePublicBase 配置项
- **文件**: `taskAuth/src/config.go`
- **内容**: 在 Config 结构体新增 `GitServicePublicBase string` 字段，从环境变量 `TASKAUTH_GIT_SERVICE_PUBLIC_BASE` 或 YAML `auth/task-auth` → `gitServicePublicBase` 加载
- **验证**: `go build ./...` 通过

### Task 1.2: 注册 EndSession 路由
- **文件**: `taskAuth/src/handlers.go`
- **内容**: 在 `registerOIDCRoutes` 中新增 `mux.HandleFunc("GET /api/oidc/endsession", handleOidcEndSession)`
- **验证**: `go build ./...` 通过

### Task 1.3: 实现 handleOidcEndSession
- **文件**: `taskAuth/src/oidc_handlers.go`
- **内容**:
  1. 从请求提取 token (Authorization header / token cookie)
  2. 调用 `deleteTokenByKey(key)` 销毁 token
  3. 清除 userId cookie (Set-Cookie: Max-Age=0)
  4. 使用 `domain.SessionTermination` 构造 GitLab sign_out URL
  5. 302 重定向至 GitLab sign_out
- **验证**: 单元测试通过

### Task 1.4: 更新 OIDC Discovery
- **文件**: `taskAuth/src/oidc_handlers.go` (handleOidcDiscovery)
- **内容**: 在 discovery JSON 中添加 `"end_session_endpoint": cfg.Issuer + "/api/oidc/endsession"`
- **验证**: `curl /.well-known/openid-configuration | jq .end_session_endpoint` 返回正确 URL

### Task 1.5: 编写 EndSession handler 测试
- **文件**: `taskAuth/src/oidc_handlers_test.go`
- **内容**:
  - TestHandleOidcEndSession_ClearsCookie: 验证 userId cookie 被清除
  - TestHandleOidcEndSession_DeletesToken: 验证 customtoken 记录被删除
  - TestHandleOidcEndSession_RedirectsToGitLab: 验证 302 Location 指向 GitLab sign_out
  - TestHandleOidcEndSession_NoTokenNoError: 无 token 时仍正常重定向
- **验证**: `go test ./... -v -run TestHandleOidcEndSession` 全部 PASS

---

## Increment 2: 主应用服务端登出

### Task 2.1: 实现 Django logout() 视图
- **文件**: `task2app/Saas_project/frontend_app/views/auth_views.py`
- **内容**: 补全 `logout()` 函数体:
  1. `auth_logout(request)` 销毁 Django session
  2. 构造 HttpResponseRedirect('/auth/login/')
  3. `response.delete_cookie('userId')` + `response.delete_cookie('sessionid')`
  4. 返回 response
- **验证**: 单元测试 — 登出后访问 `/api/accounts/profile/` 返回 401

### Task 2.2: 添加 GIT_SERVICE_PUBLIC_BASE 配置
- **文件**: `task2app/Saas_project/saas_project/settings.py` 或对应 settings 文件
- **内容**: 新增 `GIT_SERVICE_PUBLIC_BASE` 配置项，默认值从环境变量或 port_config.json 读取
- **验证**: `python -c "from django.conf import settings; print(settings.GIT_SERVICE_PUBLIC_BASE)"` 输出正确值

### Task 2.3: 修复 UserViewSet.logout() API
- **文件**: `task2app/Saas_project/accounts/views/user_views.py`
- **内容**: 替换 `_routed_via_gateway()` (501) 为:
  1. 提取 Authorization header 中的 token
  2. `try: forward_to_taskauth('POST', '/api/accounts/users/logout/', ...)` 转发删除 taskAuth token
  3. `except: pass` (降级 — taskAuth 不可达不阻断本地登出)
  4. `auth_logout(request)` 销毁 Django session
  5. 返回 `{'detail': 'ok', 'slo_redirect_url': f'{GIT_SERVICE_PUBLIC_BASE}/users/sign_out'}`
- **验证**: 单元测试 — 返回 200 + slo_redirect_url 字段

### Task 2.4: 编写 logout API 测试
- **文件**: `task2app/Saas_project/accounts/view_test/UserViewSet_logout_test.py`
- **内容**:
  - test_logout_destroys_session: 登出后 session 无效
  - test_logout_returns_slo_redirect_url: 响应包含 slo_redirect_url
  - test_logout_graceful_when_taskauth_down: taskAuth 不可达时仍返回 200
  - test_logout_clears_cookies: userId 和 sessionid cookie 被清除
- **验证**: `pytest accounts/view_test/UserViewSet_logout_test.py -v` 全部 PASS

---

## Increment 3: 前端 SLO 重定向

### Task 3.1: 更新 handleLogout() — SLO 重定向
- **文件**: `task2app/front_project/app/src/components/Navbar.logic.vue`
- **内容**: 在 `handleLogout()` 成功回调后:
  1. 清除 localStorage、userId cookie、window.currentUser
  2. 如果 `resp.data.slo_redirect_url` 存在:
     - 构造 `slo_redirect_url + '?redirect_uri=' + encodeURIComponent(window.location.origin + '/auth/login/')`
     - `window.location.href = sloUrl`（完整页面跳转，执行重定向链）
  3. 否则直接跳转 `/auth/login/`
- **验证**: 手动测试 — 点击退出后浏览器 URL 经过 GitLab sign_out 最终到 /auth/login/

### Task 3.2: 容错 — API 失败时仍清除本地状态
- **文件**: `task2app/front_project/app/src/components/Navbar.logic.vue` (同 Task 3.1)
- **内容**: `catch` 块中清除所有本地状态后跳转 `/auth/login/`
- **验证**: 模拟 API 返回 500，确认用户仍被带到登录页

---

## Increment 4: Playwright E2E 测试

### Task 4.1: Bug 复现测试（基线）
- **文件**: `gitService/playwright/tests/oidc-slo-logout-bug-repro.playwright.test.js`
- **内容**:
  1. 登录主应用 (email + password)
  2. 新标签页打开 GitLab → taskAuth SSO 登录
  3. 验证 GitLab Dashboard 可见（已登录）
  4. 切回主应用 → 点击退出
  5. 验证主应用回到登录页
  6. **刷新 GitLab 页面 → 断言仍为 Dashboard（Bug 确认: 未登出）**
- **验证**: `npx playwright test oidc-slo-logout-bug-repro` — 测试通过（确认 bug 存在）

### Task 4.2: SLO 修复验证测试
- **文件**: `gitService/playwright/tests/oidc-slo-logout-sync.playwright.test.js`
- **内容**:
  1. 登录主应用 → 验证跳转至 /projects/
  2. 新标签页打开 GitLab → taskAuth SSO 登录
  3. 验证 GitLab 已登录
  4. 切回主应用标签页 → 点击退出
  5. **等待重定向链完成（最终 URL = /auth/login/）**
  6. 切回 GitLab 标签页 → 刷新
  7. **断言: GitLab 显示登录页（已登出）**
  8. 断言: GitLab 页面出现 sign_in 文本
- **验证**: `npx playwright test oidc-slo-logout-sync` — 测试 PASS（修复生效）

### Task 4.3: Playwright 配置
- **文件**: `gitService/playwright/tests/oidc-slo-config.js` (或更新现有 playwright.config.js)
- **内容**: 测试常量 — `MAIN_APP_URL`, `GITLAB_URL`, `TEST_EMAIL`, `TEST_PASSWORD`
- **验证**: 测试文件可正确导入配置

---

## 执行顺序

```
Increment 1 (taskAuth EndSession)
  Task 1.1 → 1.2 → 1.3 → 1.4 → 1.5
    ↓
Increment 2 (Main App Logout)
  Task 2.1 → 2.2 → 2.3 → 2.4
    ↓
Increment 3 (Frontend SLO)
  Task 3.1 → 3.2
    ↓
Increment 4 (Playwright E2E)
  Task 4.1 → 4.2 → 4.3
```

每个 Increment 内部可以并行执行独立任务，但 Increment 之间存在依赖：
- Increment 2 依赖 Increment 1（需要 taskAuth EndSession 端点存在）
- Increment 3 依赖 Increment 2（需要 API 返回 slo_redirect_url）
- Increment 4 依赖 Increment 1-3（完整链路就绪）

## 文件影响范围

| 文件 | 操作 | Increment |
|------|------|-----------|
| `taskAuth/src/config.go` | 修改 — 新增字段 | 1 |
| `taskAuth/src/handlers.go` | 修改 — 新增路由 | 1 |
| `taskAuth/src/oidc_handlers.go` | 修改 — 新增 handler + discovery | 1 |
| `taskAuth/src/oidc_handlers_test.go` | 新增 — handler 测试 | 1 |
| `taskAuth/domain/session_termination.go` | **已存在** (DDD 步骤已生成) | — |
| `taskAuth/domain/customtoken_repository.go` | **已存在** (DDD 步骤已生成) | — |
| `task2app/.../auth_views.py` | 修改 — 补全 logout() | 2 |
| `task2app/.../settings.py` | 修改 — 新增配置 | 2 |
| `task2app/.../user_views.py` | 修改 — logout API | 2 |
| `task2app/.../UserViewSet_logout_test.py` | 新增 — API 测试 | 2 |
| `task2app/.../Navbar.logic.vue` | 修改 — SLO redirect | 3 |
| `gitService/playwright/tests/oidc-slo-logout-bug-repro.playwright.test.js` | 新增 | 4 |
| `gitService/playwright/tests/oidc-slo-logout-sync.playwright.test.js` | 新增 | 4 |
