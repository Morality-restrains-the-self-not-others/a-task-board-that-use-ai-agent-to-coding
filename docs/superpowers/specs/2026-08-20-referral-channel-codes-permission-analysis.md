# 角色权限 — 多渠道推荐码

- **Date:** 2026-08-20
- **Design:** `docs/superpowers/specs/2026-08-20-referral-channel-codes-design.md`

## 角色

| 角色 | 能力 |
|------|------|
| 已登录用户（本人） | 列出/创建/禁用自己的渠道码；查看自己的分渠道统计 |
| 被推荐人（注册访客） | 仅使用 `accessCode` 注册；无渠道管理 |
| 内部服务 taskAuth | `POST /api/internal/referral/bind-from-code/`（密钥） |
| 内部服务 taskReferral | `POST /api/internal/taskbill/referral/sync-edge/`（密钥） |
| 超管 | 既有审批/业绩接口不变；本增量不开放跨用户渠道读写 |

## 新/改端点授权

| 方法 | 路径 | 授权 | 拒绝 |
|------|------|------|------|
| GET | `/api/referral/channels/` | `X-User-Id` 本人 | 401 |
| POST | `/api/referral/channels/` | 本人；最多 20；名称 1–32 | 401 / 400 / 409 名称冲突 |
| POST | `/api/referral/channels/code/{code}/disable/` | 本人且非默认渠道 | 401 / 403 / 404 |
| GET | `/api/referral/stats/user_id/{userId}/` | 认证用户；**只返回调用者自己的数据**（忽略路径 userId 越权） | 401 |
| POST | `/api/internal/referral/bind-from-code/` | internal secret | 403 |
| POST | `/api/internal/taskbill/referral/sync-edge/` | bill internal secret；`commission_eligible` 省略=false | 403 |

## PDP / 租户

渠道码按 **user_id** 隔离，不属于租户 RBAC 资源组。分账仍记 `referrer_tenant_id`（推荐人公司账户）。无新 RequireRegion。

## 前端

创建渠道为写操作：同步门闩 + `Idempotency-Key`。复制链接 `Anti-Replay-OK: ui-only clipboard`。禁用渠道须确认文案，不静默 replace。
