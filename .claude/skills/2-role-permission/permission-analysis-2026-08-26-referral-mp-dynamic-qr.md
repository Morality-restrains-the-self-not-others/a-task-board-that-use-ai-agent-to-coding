# 角色权限分析：推荐资格服务号动态 scene 码

- **Date:** 2026-08-26
- **Design:** `2026-08-26-referral-mp-dynamic-qr-ticket-design.md`
- **Verdict:** 绿灯 ✅

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST `/api/auth/wechat/mp/follow-qr/` | 已登录用户 | Resource（本人） | write ticket self | token + `X-User-Id` | ✅ | 禁止 body 指定 user_id；只为会话用户签发；Idempotency-Key 必填 |
| GET `/api/auth/wechat/mp/follow-status/` | 已登录用户 | Resource（本人） | read self | token + `X-User-Id` | ✅ | 扩展 ticket 字段；禁止返回他人 user_id/unionId/openid |
| GET/POST `/api/auth/wechat/mp/callback/` | 微信服务器 | System | write identity/ticket | 签名/AES | ✅ | public；禁止登录 0.5/s 限流；验签失败 403 |
| `auth_wechat_mp_follow_ticket` | 系统 + 本人签发 | Resource | write | 票 user_id vs unionId 占用 | ✅ | 冲突不抢绑；conflict_owner 仅服务端 |
| `wechat_identity` mp 行 | 系统（回调） | Resource | write | 票用户或 v112 unionId 路径 | ✅ | 禁止关注建号 |
| 模拟登录 | 超管 | Resource（被模拟用户） | 签发/查询 | `X-User-Id` 为目标用户 | ✅ | 日志带 impersonator 字段；不得用模拟者 id 签发票 |

## 角色与权限建模

无新 RBAC 角色。不登记租户 page/region（个人资料推荐页，非租户控制台）。

检查路径：L1 APISIX token（follow-qr / follow-status）；callback public + 验签。L4 `resolveUserIDFromRequest` 仅当前用户。

## 安全审查结论

- [x] **IDOR**：follow-qr / follow-status 无路径资源 id；票按 `X-User-Id` 过滤。
- [x] **权限提升**：无 PATCH 改他人绑定。
- [x] **跨租户泄露**：身份表平台级；冲突文案不含对方账号。
- [x] **403 vs 404**：未登录 401；微信验签失败 403。
- [x] **user_id 注入**：禁止客户端传入 user_id。
- [x] **敏感操作**：冲突不改绑；审计事件 `WECHAT_IDENTITY_CONFLICT`。
- [x] **伪造回调**：无 token 的 POST 必须 WeChat signature / AES。
- [x] **PII**：日志只记 openid/unionid 指纹；不打 ticket 明文可降级为指纹（实施时 ticket 仅存库）。
- [x] **模拟登录**：签发与查询使用被模拟用户；日志含 impersonating 字段。

## 权限测试清单

| 测试场景 | 角色 | 操作 | 预期 |
|----------|------|------|------|
| 未登录签发动态码 | anonymous | POST follow-qr | 401 |
| 已登录签发 | 本人 | POST follow-qr | 200，票 user_id=自己 |
| 缺 Idempotency-Key | 本人 | POST follow-qr | 400 |
| unionId 属他人 SCAN | 微信回调 | POST callback | 200 success；mp 仍属原用户；当前用户 follow-status conflict |
| follow-status 冲突响应 | 本人 | GET | 无他人 user_id/unionId/openid |
| 伪造回调 | 匿名 | POST 坏签名 | 403 |

## 风险评级

| 项 | 等级 | 缓解 |
|----|------|------|
| 账号夺取（扫别人的码抢绑） | 高（已缓解） | 先匹配 temp_id，再检查 unionId 占用，冲突不 upsert |
| 冲突文案泄露对方身份 | 中（已缓解） | 仅稳定 conflict_code + 通用 message |
| 微信创码失败回退静态码 | 中（已禁止） | 503 + data-traceId |

## 总结清单

- follow-qr：仅会话用户，token 路由
- callback：public 验签，按票绑定或 conflict
- follow-status：本人只读 + 安全冲突文案
