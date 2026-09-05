# 角色权限分析：登录后手机号验证引导

- 日期：2026-08-25
- 关联设计：2026-08-25-login-phone-verify-prompt-design.md

## 结论

本期**不新增、不修改**鉴权端点。弹窗是已认证会话上的前端 UX；「去验证」落到用户自己的 `/profile/`，绑定仍走既有 `bind-phone`（仅操作当前 token 对应用户）。

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| 登录成功后读 `user.login_methods` | 刚登录的本人 | User | read | 登录响应只含当前用户 | ✅ | 前端勿把 identifier 打进日志 |
| GET `/api/accounts/users/profile/`（微信路径已有） | 已认证用户 | User | read | `resolveUserIDFromRequest` | ✅ | 不新增调用或复用 identity 返回值 |
| POST bind-phone（存量） | 已认证用户 | User | write | requireAuthenticatedUser + SMS | ✅ | 本期不改 |
| `/profile/#rg=profile.phone_binding` | 已认证用户 | User | navigate | 路由既有登录守卫 | ✅ | 深链不得带他人 user id |
| 管理员登录 | 平台管理员 | System | — | `adminLogin` 跳过弹窗 | ✅ | 避免打断控制台工作流 |
| 模拟登录 | 操作员+目标用户 | User | — | 检测 impersonator backup 跳过 | ✅ | 避免把操作员引导去绑目标用户手机 |

## 新角色

无。

## IDOR

「去验证」固定 `/profile/`（当前用户资料），不接受 URL 中的他人 `user_id`。

## 审计

弹窗本身不写审计日志。绑定成功沿用存量 bind-phone 日志。禁止日志含完整手机号。
