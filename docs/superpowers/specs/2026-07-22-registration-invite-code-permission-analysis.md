# 注册邀请码 — 角色权限分析

- **日期**: 2026-07-22
- **设计**: `2026-07-22-registration-invite-code-design.md`

## 端点 × 角色

| 端点 | 匿名 | 登录用户 | Superuser |
|---|---|---|---|
| GET `/api/public/registration-invite-policy/` | ✅ 只读摘要 | ✅ | ✅ |
| GET/PUT `/api/system-admin/registration-invite-policy/` | ❌ | ❌ | ✅ |
| GET `/api/system-admin/registration-invite-relations/` | ❌ | ❌ | ✅ |
| POST/GET `.../registration-invite-codes*` | ❌ | ✅（仅本人数据） | ✅（本人；Admin 用 relations） |
| email/phone_register + invite_code | ✅（公开注册） | n/a | n/a |

## 强制规则

1. Admin 接口：`X-User-Id` + `accounts_super_admin`/`is_superuser`，否则 403。
2. 用户列表/申请：仅操作自身 `issuer_user_id`；禁止通过 query 指定他人。
3. 核销仅发生在注册成功写用户之后、响应之前；失败回滚码状态（同事务）。
4. 不信任客户端伪造的 `issuer_user_id` / `redeemed_by`。

## CRG 触点

- graph_status: sparse；敏感边：register → redeem、admin PUT policy。
- 风险：直连 taskAuth 端口伪造 `X-User-Id` — 运维约束端口不公网（既有）。
