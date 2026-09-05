# 价值流 — 管理端赠送页修改 VIP 等级

Mapping the approved design into a value stream.

- **日期**: 2026-08-18
- **设计**: `docs/superpowers/specs/2026-08-18-admin-grant-membership-tier-design.md`

## Related Value Streams

- `2026-08-18-admin-grant-gitlab-region-value-stream.md`：同页赠送 GitLab。本流是同页扩展，不撤销区域约束。
- 购买 VIP1 门槛（`checkMembershipPurchasePermission`）：本流让管理员可授予 VIP1，使租户能走购买路径。

## 价值增量（最小可交付）

单一垂直切片：

1. **管理员把租户设为 VIP1（可同时赠送资源）**  
   打开赠送页 → 选租户 → 看到当前等级 → 选择 VIP1 →（可选）填资源 → 确认 → `billing_membership.tier=vip1` 且锁定。

步骤：

| 步 | 角色 | 系统 | 产出 |
|----|------|------|------|
| 加载当前等级 | 系统管理员 | GET membership | 展示 normal/vip1 |
| 选择目标等级 | 系统管理员 | taskFE | `membership_tier` |
| 提交 | 系统管理员 | `adminGrantResourcesWithMembership` | 等级 + 可选配额 |
| 之后购买 GitLab | 租户 | `createOrder` | VIP1 可通过权限校验 |
| 累计消费不再覆盖 | 系统 | `syncMembershipConsumption` | locked 时不自动升级 |

## 不做

- 不新增 Kafka、不改 100 元自动升级阈值、不开放租户自助改等级。
