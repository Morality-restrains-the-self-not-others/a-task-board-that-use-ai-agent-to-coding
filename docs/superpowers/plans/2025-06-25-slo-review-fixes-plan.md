# 实施计划: SLO 代码审查修复

> 输入:
> - 设计: `docs/superpowers/specs/2025-06-25-slo-review-fixes-design.md`
> - 价值流: `docs/superpowers/plans/2025-06-25-slo-review-fixes-value-stream.md`
> - NFR: `docs/superpowers/plans/2025-06-25-slo-review-fixes-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2025-06-25-slo-review-fixes-ddd-audit.md`

## 任务总览

| Increment | 任务数 | 主题 |
|-----------|--------|------|
| 1 | 4 | 🔴 EndSession 安全加固（开放重定向 + 测试） |
| 2 | 4 | 🟡 Cookie 属性 + 异常日志 |
| 3 | 6 | 🟡+⚪ 死代码清理 + 代码质量 |
| 4 | 4 | 🟡 测试修复 |

---

## Increment 1: EndSession 安全加固

### 1.1 新增 redirect_uri 白名单验证函数
- [ ] 文件: `taskAuth/src/oidc_handlers.go`
- [ ] 新增 `isValidPostLogoutRedirectURI(uri string) bool` 函数
- [ ] 解析 URL → 检查 scheme (仅 http/https) → 构建 allowlist (GatewayPublicBase + GitServicePublicBase) → host 匹配
- [ ] 若 uri 为空，返回 true（空值合法）
- [ ] 命令: `cd /tmp/ram-work/taskAuth && bash run.sh build`

### 1.2 在 handleOidcEndSession 中调用验证
- [ ] 文件: `taskAuth/src/oidc_handlers.go`
- [ ] 在构造 redirectURL 前调用 `isValidPostLogoutRedirectURI(postLogoutRedirectURI)`
- [ ] 若验证失败：fallback 至默认登录页 (GatewayPublicBase + "/auth/login/")
- [ ] 命令: `cd /tmp/ram-work/taskAuth && bash run.sh build`

### 1.3 更新 Go 单元测试
- [ ] 文件: `taskAuth/src/oidc_endsession_test.go`
- [ ] 更新 `TestHandleOidcEndSession_RedirectsToGitLab`: 期望 Location 为 "/auth/login/"
- [ ] 更新 `TestHandleOidcEndSession_WithPostLogoutRedirect`: 期望 Location 为 post_logout_redirect_uri 值
- [ ] 重命名 `TestHandleOidcEndSession_NoGitServiceBase_FallsBackToGateway` → `TestHandleOidcEndSession_WithPostLogoutRedirect_Passthrough`
- [ ] 新增 `TestHandleOidcEndSession_RejectsExternalRedirectURI`: 传入 `post_logout_redirect_uri=https://evil.com/`，断言 Location 不含 evil.com
- [ ] 命令: `cd /tmp/ram-work/taskAuth && go test ./src/ -run TestHandleOidcEndSession -v`

### 1.4 部署 taskAuth
- [ ] 构建: `cd /tmp/ram-work/taskAuth && bash run.sh build`
- [ ] 重启: `curl -X POST :9999/api/restart -d '{"name":"task-auth","session_id":"..."}'`
- [ ] 验证: `curl -D - "http://183.250.1.132:18081/api/oidc/endsession?post_logout_redirect_uri=https://evil.com/"` → 期望 Location 不含 evil.com

---

## Increment 2: Cookie 属性 + 异常日志

### 2.1 taskAuth: Set-Cookie 添加 SameSite 属性
- [ ] 文件: `taskAuth/src/oidc_handlers.go`
- [ ] 所有 `http.SetCookie` 调用追加 `SameSite: http.SameSiteLaxMode`
- [ ] userId cookie: `SameSite: http.SameSiteLaxMode`
- [ ] _gitlab_session cookie: `SameSite: http.SameSiteLaxMode`
- [ ] 命令: `cd /tmp/ram-work/taskAuth && bash run.sh build`

### 2.2 Django: 异常日志记录
- [ ] 文件: `task2app/Saas_project/accounts/views/user_views.py`
- [ ] 在 `except Exception: pass` 处替换为 `except Exception: logger.warning("taskAuth unreachable during logout", exc_info=True)`
- [ ] 确认 `logger` 在文件顶部已导入（行 65: `logger = logging.getLogger(__name__)`）

### 2.3 Django: delete_cookie 添加 samesite 参数
- [ ] 文件: `task2app/Saas_project/accounts/views/user_views.py`
- [ ] `response.delete_cookie('userId')` → 添加 `samesite='Lax'`
- [ ] `response.delete_cookie('sessionid')` → 添加 `samesite='Lax'`

### 2.4 部署 Django
- [ ] 重启: `curl -X POST :9999/api/restart -d '{"name":"saas-backend","session_id":"..."}'`
- [ ] 验证: 查看日志确认 WARNING 级别日志格式正确

---

## Increment 3: 死代码清理 + 代码质量

### 3.1 删除 SessionTermination 死代码
- [ ] 文件: 删除 `taskAuth/domain/session_termination.go`
- [ ] 文件: 删除 `taskAuth/domain/session_termination_test.go`
- [ ] 验证: `grep -r "SessionTermination\|GitLabSignOutURL" taskAuth/src/` → 零结果

### 3.2 删除 CustomTokenRepository 死代码
- [ ] 文件: 删除 `taskAuth/domain/customtoken_repository.go`
- [ ] 验证: `grep -r "CustomTokenRepository" taskAuth/` → 零结果

### 3.3 前端: URL 构造安全化
- [ ] 文件: `task2app/front_project/app/src/components/Navbar.logic.vue`
- [ ] 替换字符串拼接: `const u = new URL(data.slo_redirect_url); u.searchParams.set('post_logout_redirect_uri', loginUrl); window.location.href = u.toString()`
- [ ] 命令: Vite HMR 自动更新

### 3.4 前端: 函数去重
- [ ] 文件: `task2app/front_project/app/src/components/Navbar.logic.vue`
- [ ] 提取 `resetAuthState()` 函数（合并 `setLoggedOutUser` 和 `clearLocalAuthState` 的共同逻辑）
- [ ] `setLoggedOutUser` 和 `clearLocalAuthState` 调用 `resetAuthState()`

### 3.5 APISIX: 路由 methods 约束
- [ ] 文件: `taskGateway/routes/routes.yaml`
- [ ] `django-logout` 路由添加 `methods: [POST]`
- [ ] 命令: `cd /tmp/ram-work/taskGateway && python3 scripts/routes-to-apisix.py`

### 3.6 部署 APISIX
- [ ] 重启: `curl -X POST :9999/api/restart -d '{"name":"task-gateway","session_id":"..."}'`

---

## Increment 4: 测试修复

### 4.1 Playwright SLO-003: 增加 GitLab SSO 步骤
- [ ] 文件: `gitService/playwright/tests/oidc-slo-logout-sync.playwright.test.js`
- [ ] Step 2 新增: 导航到 GitLab → 点击 taskAuth SSO → 确认登录成功 → 记录 _gitlab_session cookie
- [ ] 断言改为: `expect(gitlabSessionBefore).toBeDefined(); expect(sessionChanged).toBe(true)`
- [ ] 命令: `npx playwright test oidc-slo-logout-sync.playwright.test.js --reporter=list`

### 4.2 运行 Go 单元测试
- [ ] 命令: `cd /tmp/ram-work/taskAuth && go test ./... -v 2>&1 | tail -30`
- [ ] 期望: 全部 PASS，包括新增/更新的 EndSession 测试

### 4.3 运行 Playwright E2E 测试
- [ ] 命令: `npx playwright test oidc-slo-logout-sync.playwright.test.js --reporter=list`
- [ ] 期望: 3 tests PASS

### 4.4 运行 DDD 合规检查
- [ ] 命令: `python3 runAll/scripts/ci/check_ddd_bdd_compliance.py`
- [ ] 期望: PASS

---

## 依赖关系

```
Increment 1 (安全加固) ──→ Increment 2 (Cookie + 日志)
         │                         │
         └────── Increment 3 (代码质量) ──────┘
                        │
                        └── Increment 4 (测试修复)
```
