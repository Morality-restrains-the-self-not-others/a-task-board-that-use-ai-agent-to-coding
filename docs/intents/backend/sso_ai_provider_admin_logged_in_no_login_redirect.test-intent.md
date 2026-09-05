# 测试意图：镜像市场管理 SSO 经 /api/ 网关认证

## 覆盖范围

| ID | 场景 | 预期 |
|----|------|------|
| T1 | `/api/accounts/sso/ai-provider/admin/` + 合法网关头 | middleware 设置 `request.user` |
| T2 | 同路径仅有 `userId` cookie、无网关头 | **不**认证（无 cookie 兜底） |
| T3 | 前端 href / Playwright 使用 `/api/accounts/sso/...` | 路径一致 |
| T4 | 旧 `/accounts/sso/...` | 不再由 Django SSO 视图处理 |

## 自动化

- 单元：`task2app/Saas_project/tests/test_gateway_auth_middleware_sso_api_path.py`
- E2E：`playwright/front_project/tests/SystemAdmin.sso-ai-provider-admin.playwright.test.js`
- E2E：`task2app/playwright/saas_ai_provider/tests/django8010-sso-*.playwright.test.js`

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-07-13 | 初版 cookie 回退测例 |
| 2026-07-13 | 改为网关头测例；删除 cookie 兜底测例 |
