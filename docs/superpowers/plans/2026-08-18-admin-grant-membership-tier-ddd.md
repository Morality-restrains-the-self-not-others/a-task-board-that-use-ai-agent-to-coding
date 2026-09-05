# DDD — 管理端赠送页修改 VIP 等级

- **日期**: 2026-08-18
- **NFR**: `docs/superpowers/plans/2026-08-18-admin-grant-membership-tier-nfr-clarification.md`

taskBill 仍为 `main` 包过程式服务。不新建 `domain/` 包。实现落在 `Membership` / `adminGrantResourcesWithMembership`。

## 限界上下文

计费 / 租户会员（owner：taskBill）。

## 聚合

**TenantMembership**（每租户一行 `billing_membership`）

- 实体属性：`tenant_id`, `tier` (`normal`|`vip1`), `cumulative_consumption_cents`, `admin_tier_locked`, `upgraded_at`
- 不变量：
  - 管理员设等级 → `admin_tier_locked=true`
  - 自动升级仅当 `!admin_tier_locked && tier==normal && cumulative>=10000`

**ResourceGrantBatch**（已有）：本次可选与会员写同一事务。

## 领域服务

`adminGrantResourcesWithMembership`：可选调级 + 既有赠送。  
`syncMembershipConsumption`：尊重 lock。

## 端口

不新增；`*sql.DB`。

## 领域事件

无新增 Kafka。书面例外见 B-049c。日志事件名 `membership_admin_tier_set`。

## 幂等

赠送仍用 `billing_idempotency_key`。调级目标态：同一 tier 重复写入成功。
