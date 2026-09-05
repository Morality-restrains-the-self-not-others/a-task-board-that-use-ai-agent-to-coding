# 实施计划: SSO 401 Fix — Forward-Auth Cookie Bridge

> 来源:
> - 设计文档: `docs/design/sso-401-fix-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-29-sso-401-forward-auth-cookie-bridge-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-29-sso-401-forward-auth-cookie-bridge-nfr-clarification.md`
> - DDD: Skip (无新领域概念)

---

## Task 1: Go 单元测试 — forward-auth userId cookie 认证

**文件:** `taskAuth/src/gateway_forward_auth_test.go`
**描述:** 新增测试用例验证 `handleGatewayForwardAuth` 在仅有 `userId` cookie 时返回 200

**测试用例:**
- [ ] userId cookie → 用户存在且活跃 → 200 + X-User-Id
- [ ] userId cookie → 用户不存在 → 401
- [ ] userId cookie → 用户未激活 → 401
- [ ] 无 cookie 无 token → 401
- [ ] Authorization: Token xxx → 200 (回归)
- [ ] Token 优先级高于 cookie (回归)

**命令:** `cd taskAuth && go test -v -run TestGatewayForwardAuth ./...`

---

## Task 2: 修改 handleGatewayForwardAuth

**文件:** `taskAuth/src/gateway_forward_auth.go` (第 17-25 行)
**变更:** 替换 3 行为 1 行
```go
// Before:
tokenKey := tokenFromRequest(r)
if tokenKey == "" {
    w.WriteHeader(http.StatusUnauthorized)
    return
}
userID, err := resolveTokenUserID(tokenKey)

// After:
userID, err := resolveTokenUserIDFromRequest(r)
```

---

## Task 3: 运行 Go 测试 + 编译

**命令:** 
```bash
cd taskAuth && go test ./... && go build -o bin/taskAuth .
```

---

## Task 4: Playwright E2E — SSO 跳转验证

**文件:** `playwright/front_project/tests/SystemAdmin.sso-ai-provider-admin.playwright.test.js`
**描述:** 模拟完整登录 → 点击 SSO 链接 → 验证跳转成功（302 → Ai Provider）

**测试用例:**
- [ ] 管理员登录后点击 SSO → 新标签页不返回 401
- [ ] 验证响应包含 `sso_bridge` JWT (302 redirect)

**命令:** `node playwright/front_project/tests/SystemAdmin.sso-ai-provider-admin.playwright.test.js`

---

## Task 5: 回归检查

- [ ] 所有现有 Go 测试通过 (`go test ./...`)
- [ ] 现有 Token auth 请求不受影响
- [ ] Value stream YAML 校验通过 (`cd valueStream && go test ./...`)
