# 设计：管理端赠送显式数量（避免误赠订单）

- **日期**: 2026-08-23
- **页面**: `/system-admin/grant-points`；租户侧 `/tenant/:id/billing/orders/` 展示误赠订单
- **状态**: accepted（goal-mode）
- **架构变更**: 否（无新服务/表/端点）

## Context

订单 `ORD-20260822-877397588196749312-879129787220656128`（id `879129787220656128`）：

- `payment_method=admin_grant`，金额 0，`paid_at=2026-08-22 22:27:54`
- 行项：任务帖 × 100；grant `source_kind=gift` remaining=100
- 流水：`user_id=bootstrap-admin`，描述「管理员后台赠送：任务帖 +100 帖；会员等级=vip1」

根因：赠送页 `freshLine().quantity` 预填 **100**。管理员只想设 VIP1 时，默认行仍是合法 `quantity>=1`，与 `membership_tier` 一并提交，后端按设计为赠送资源建零元订单。该租户 08-18 已有「新用户礼包」100 帖，此次为重复赠送。

点击订单号本身不会触发赠送（仅为详情 GET；近期日志为 bootstrap-admin 模拟登录查看）。

## Decision

1. 数量默认空；未填 ≥1 的行不进入 `resources`。
2. 提交区展示核对文案（是否建赠送订单 / 是否改 VIP）。
3. POST 使用 `createClickGuard` + `Idempotency-Key`（与 body `idempotency_key` 同值）；后端同时认 header。
4. **不回滚**该历史订单（100 帖尚未消耗）；防再发即可。

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 二次确认弹窗 | 能防误触，但测例面更大；空默认已切断根因 |
| 自动冲正该单 | 资金/配额冲正需产品确认；剩余未消耗，先防再发 |
| 后端拒绝「VIP+默认 100」 | 无法区分有意赠 100 帖 |

## Consequences

- 只改 VIP 与既有 `TestAdminGrantMembershipOnlyNoOrder` 对齐。
- 有意赠 100 帖须手填数量。
