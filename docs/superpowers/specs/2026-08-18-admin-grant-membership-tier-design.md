# 管理端赠送页可修改租户 VIP 等级

- **日期**: 2026-08-18
- **作者**: cursor
- **迭代**: admin-grant-membership-tier
- **状态**: accepted（goal-mode 自动采用）
- **意图**: `docs/intents/backend/admin_grant_membership_tier.intent.md`（B-049c）
- **python_api_approval**: n/a（扩展既有 Go `taskBill` 赠送 API + Vue 赠送页）
- **架构**: **不升版**。沿用 v85；无新服务/新库/新通信方式。
- **ADR**: `No-ADR: trivial tech choice, no architectural impact`（扩展既有 `billing_membership` + 赠送页）

---

## 0. 完成标准（SMART）

1. `/system-admin/grant-points/` 在选中租户后展示当前 VIP（`normal` / `vip1`），并提供下拉：**不修改 / 普通会员 / VIP1**。
2. 仅改 VIP、仅赠送资源、或两者同时提交均可；未选等级时行为与现赠送一致。
3. `POST /api/tenant/{tid}/billing/accounts/admin_grant_points/` 可选字段 `membership_tier`；非法值 400；合法值写入 `billing_membership.tier` 且 `admin_tier_locked=1`。
4. 管理员锁定后，`syncMembershipConsumption` **不得**因累计消费把 `normal` 自动升回 `vip1`。
5. 租户自助购买路径不可调用该字段；仅系统管理员赠送页。
6. 回归：既有 GitLab 区域赠送、任务帖赠送、推荐佣金 `adminGrantResources` 不变。

## 1. 问题

购买 GitLab 磁盘/流量要求 VIP1。管理员只能等租户累计消费满 100 元自动升级，无法在「赠送资源」页把指定租户调成 VIP1（或纠错降回普通）。

## 2. 方案（已选定）

**在赠送页增加独立「VIP 等级」区块（不是资源行）**；同一 POST 可选带 `membership_tier`。

拒绝的替代：

| 方案 | 拒绝原因 |
|------|----------|
| 做成 `resource_type=membership` 行 | 数量/过期/区域语义不适配 |
| 新独立 API + 新菜单 | 用户要求做在赠送资源里；权限面已覆盖 |
| 只允许升级不允许降级 | 「修改」含纠错；降级须锁定以免下次支付立刻升回 |
| 不锁定、靠文档说明自动升级会覆盖 | 运维不可预期 |

### 2.1 API（扩展既有）

`POST /api/tenant/{tid}/billing/accounts/admin_grant_points/`

```json
{
  "resources": [],
  "membership_tier": "vip1",
  "membership_reason": "开通 GitLab 购买"
}
```

- `membership_tier`：省略或不改 = 不碰会员；`normal` | `vip1` 为合法值。
- `resources` 可空，当且仅当带了合法 `membership_tier`。
- 内部 `admin-grant-resources` / 推荐佣金 **不**传该字段（避免礼包误改 VIP）。
- 错误：`invalid membership tier`，HTTP 400。

### 2.2 落库

- `billing_membership.tier` = 所选等级。
- `admin_tier_locked` TINYINT NOT NULL DEFAULT 0；管理员写入时置 1。
- `vip1` 时更新 `upgraded_at`。
- VIP-only：记 `billing_transaction`（`points_source_type=admin_grant`），**不**建空订单。
- 与资源同行提交：同一事务内改等级 + 既有赠送逻辑。

### 2.3 UI

- 独立卡片，`data-testid="grant-membership-tier"`。
- 选中租户后 `GET /api/tenant/{tid}/billing/membership/` 展示当前等级。
- 允许移除全部资源行（仅改 VIP）；默认仍带一行任务帖 100，避免打断现赠送心智。
- 提交：有有效资源行 **或** 选择了等级。

### 2.4 事件

无新 Kafka。对照表书面例外见意图文档。观测：`membership_admin_tier_set`。

## 3. CRG

增量无新节点。链：`handleAdminGrantResources` → `adminGrantResourcesWithMembership` → `billing_membership`；FE `SystemAdminGrantPoints`。

## 4. 架构判断

无需新 `.puml` / `.archimate`：无新 Application_Component、无新数据所有权。
