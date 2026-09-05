# 角色权限分析：推荐资格服务号关注闸门

- **Date:** 2026-08-26
- **Design:** `2026-08-26-referral-mp-follow-gate-design.md`

## 端点权限

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET/POST `/api/auth/wechat/mp/callback/` | 微信服务器 | System | write identity | 签名/AES | ✅ | public；禁止用登录 0.5/s 限流；验签失败 403 |
| GET `/api/auth/wechat/mp/follow-status/` | 已登录用户 | Resource（本人） | read+bind self | token + X-User-Id | ✅ | 只绑定当前用户 unionid 的 pending |
| GET referral status `service_account_bound` | 已登录用户 | Resource（本人） | read | token | ✅ | 只读本人 |
| POST referral apply | 已登录用户 | Resource（本人） | write | token + 原申请校验 | ✅ 补闸门 | 未绑定 mp 拒绝 |
| `wechat_identity` mp 行 | 系统（回调） | Resource | write | unionid 锁定 | ✅ | 禁止关注事件建新用户；冲突发 WECHAT_IDENTITY_CONFLICT |
| pending 表 | 系统 | System | write | unionid PK | ✅ | 不含 user_id，避免未认证写他人 |

## 威胁

- **伪造关注回调**：无 token 的 POST 必须校验 WeChat signature（及加密模式 AES）。
- **IDOR**：follow-status 不得接受 client 传入的 unionid/openid。
- **账号夺取**：unionid 已属用户 A 时不得改绑到用户 B；走既有 conflict 路径。
- **PII**：日志只记 openid/unionid 长度或指纹，不打 token/secret。

## 新角色

无。不新增 RBAC 角色。
