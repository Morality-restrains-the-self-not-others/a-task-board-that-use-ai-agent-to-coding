# 测试意图：用户编辑页模拟登录按钮

- **对应功能意图**: `system_admin_user_impersonation.intent.md`
- **日期**: 2026-08-23

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| F1 | hasPlatformPerm('user:impersonate')=true | 编辑表单有按钮 |
| F2 | 无该权限 | 无按钮 |
| F3 | 点击 | 调用 POST impersonate，带 Idempotency-Key |
| F4 | 连点 | 第二次不发新请求（guard busy） |
| F5 | API 403 | 错误展示含 data-traceId，仍在模态框 |
| F6 | API 200 | 更新账号槽并跳转 redirect_url（非系统管理） |
| F6b | API 409 already impersonating 且 status 同目标 | 跳转 status.redirect_url，不停留弹窗 |
| F7 | status.impersonating | Navbar 横幅 + 退出按钮 |
| F8 | 退出 | POST stop，跳转系统管理用户页 |
