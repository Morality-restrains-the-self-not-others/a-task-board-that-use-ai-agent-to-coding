# NFR 澄清 — 管理端赠送页修改 VIP 等级

- **日期**: 2026-08-18
- **价值流**: `docs/superpowers/plans/2026-08-18-admin-grant-membership-tier-value-stream.md`
- **默认等级**: L2（会员写路径影响购买权限 → 幂等按 L3 审视）

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 可伸缩性 | 动作 |
|------|---------|----------|----------|------|
| `POST /api/tenant/{tid}/billing/accounts/admin_grant_points/` | `tid` = tenant_id | 是（会员按租户一行） | L1 | 保持 URL 中 tenant |
| `GET /api/tenant/{tid}/billing/membership/` | `tid` | 是 | L1 | 保持 |
| 前端 `/system-admin/grant-points/` | 无 | 管理后台单页 | L0：管理员人数极少 | 升级触发：管理页需按租户分片检索时再补 |

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| admin_grant_points POST（含 membership_tier） | 改 tier + lock；可选加配额 | 双击、网关重试 | 一次确认 = 一次赠送/一次调级 | 可选 `idempotency_key`（已有） | 相同 key 返回已有结果，不重复改配额；调级本身目标态幂等（同 tier 再写仍 ok） |
| GET membership | 无 | — | — | L0 | — |
| syncMembershipConsumption | 可能 auto vip1 | 支付成功 | 租户累计消费跨过阈值 | 条件更新 `tier='normal' AND NOT locked` | locked 跳过 |

禁止用 `tenant_id` 单独作幂等键。未传 key 时两次提交允许两次赠送（现行为）；两次同 tier 调级结果仍为目标态。

## 类别等级

| 类别 | 等级 | 说明 |
|------|------|------|
| 可伸缩性 | L1 | 按 tenant_id，每租户一行 |
| 数据一致性 | L3 | 与赠送同事务（若同时提交） |
| 安全 | L2 | 仅系统管理员；tier 白名单 |
| 可用性 | L2 | 失败 400 可重试 |
| 可观测性 | L2 | `membership_admin_tier_set` 含 from/to/locked（无 PII） |

## 质量场景

1. 刺激：`membership_tier=vip2`。响应：400，tier 不变。
2. 刺激：仅 `membership_tier=vip1`、空 resources。响应：200，tier=vip1，locked=1，无空订单。
3. 刺激：locked 的 normal 租户累计消费 ≥100 元后支付。响应：仍为 normal。
4. 刺激：未传 membership_tier 的赠送。响应：会员行不变。

## 领域模型影响

`Membership` 增加 `AdminTierLocked`。自动升级不变量：仅 `tier=normal AND NOT locked AND cumulative>=threshold`。
