# 角色权限分析：移除手机号验证码登录

- 日期：2026-08-24
- 关联设计：2026-08-24-remove-phone-code-login-design.md

## 端点权限审计

| 端点 | 变更 | 认证要求 | 权限分析 |
|---|---|---|---|
| `POST /api/auth/`（phone+code 分支） | **拒绝**（400 fail-closed） | 匿名 | 无新增权限面；匿名调用从「可登录+自动注册」降级为「拒绝」 |
| `POST /api/accounts/users/phone_register/` | 不变 | 匿名+验证码 | 无变化 |
| `POST /api/accounts/users/send_verification_code/` | 不变 | 匿名 | 无变化 |
| `POST /api/accounts/users/send_password_reset_code/` | 不变 | 匿名 | 无变化 |
| `POST /api/accounts/users/reset_password_with_code/` | 不变 | 匿名+验证码 | 无变化 |

## 结论
- 本任务为**功能删除**，不新增任何端点，不引入新角色/权限
- 删除 OTP 自动注册路径 = 匿名用户在未注册手机号上「验证码直登」能力收敛，权限面收窄（安全正向）
- admin-login / accessToken / 微信 OAuth 不受影响
- **无权限变更，本步骤通过**
