# 角色权限分析：导航栏多账号切换

**日期**: 2026-07-14  
**设计文档**: `docs/superpowers/specs/2026-07-14-navbar-multi-account-switcher-design.md`  
**状态**: 完成（goal-mode 自动）

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有/拟定检查 | 是否缺失 | 建议 |
|--------|------|----------|------|---------------|----------|------|
| POST `/api/accounts/users/activate-session/` | 持有合法 Token 的用户 | User (self via token) | write (session activate) | Token 校验 + is_active + 可选 user_id 匹配 | ✅ | 禁止无 Token；禁止用 A 的 Token 激活 B（user_id 校验） |
| localStorage `savedAccounts` | 浏览器本机用户 | Device | RW | 客户端限制；上限 5 | ✅ | XSS 面与现网 Token 一致；退出时删除槽 token |
| Navbar 切换 UI | 已登录用户 | UI | read/switch | 仅展示本机槽 | ✅ | 切换必须走 activate-session，禁止只改 cookie |
| Login upsert 槽 | 登录成功用户 | Device | write | 登录 API 已鉴权 | ✅ | add_account 不提升权限 |
| 退出当前 | 当前激活用户 | User | logout | 现有 logout | ✅ | 删除服务端当前 token + 本地槽 |

## IDOR / 串号风险

- **风险**: 攻击者改 cookie `userId` 但不改 token → API 仍以 Token 用户为准（现有行为）；UI 可能短暂不一致。
- **缓解**: 切换后以 activate-session 返回的 `user.id` 写回 cookie；AuthSessionGuard 以 `/me/` 为准。

## 新角色

无。不引入新 RBAC 角色。

## 结论

权限边界充分；无阻塞项。可进入价值流。
