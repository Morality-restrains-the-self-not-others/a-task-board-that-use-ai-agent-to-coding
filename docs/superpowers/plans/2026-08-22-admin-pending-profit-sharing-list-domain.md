# DDD — 管理员待分账队列（Billing）

- **日期**: 2026-08-22
- **设计**: `docs/superpowers/specs/2026-08-22-admin-pending-profit-sharing-list-design.md`

## Bounded Context

Billing（taskBill）。不新建服务。

## Aggregates / Entities

- **ProfitSharingRecord**（已有）：根为分账记录 `id` / `out_profit_sharing_no`；按 `order_id` 关联订单。状态机 pending → processing → finished | failed。
- 本增量不新增聚合，不新增写命令。

## Read model

- **ProfitSharingQueueItem**：列表投影。字段见设计契约。省略 `referrer_openid`、`wechat_profit_sharing_id`。

## Domain events

无新事件。写入仍由支付完成意图发布后的既有路径 `markOrderForProfitSharing`（记录插入，非本增量）。

## 仓储

- `listProfitSharingQueue(status, limit, offset)` → items + total
- 查询条件：status 集合；ORDER BY settle_after ASC, id ASC
- Owner：taskBill / `billing_profit_sharing`

## 架构变更影响

- **迭代版本**: v99 🎯 target
- **迭代名称**: admin-pending-profit-sharing-list
- **作者**: cursor
- **设计日期**: 2026-08-22 21:55
- **新增文件**（每个视图四类伴生格式）:
  - 🆕 `docs/architecture/v99-enterprise-landscape-20260822-2155-cursor.puml`
  - 🆕 `docs/architecture/v99-application-integration-20260822-2155-cursor.puml`
  - 🆕 各视图 `.diff.archimate`（增量变迁：v98→v99 + Plateau/Gap/WP）
  - 🆕 各视图 `.full.archimate`（全量拓扑）
  - 🆕 各视图 `.mermaid.md`
- **已有文件（未修改）**:
  - `docs/architecture/v98-*-20260822-1555-cursor.puml` (current)
  - v97 Git OAuth site 寻址仍为正交 target
- **变更明细**: 🟢 新增 `GET /api/system-admin/profit-sharing/`；🟡 taskFE 订单页第三 Tab

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v98 → Gap「无待分账队列」→ WP → Plateau v99；Vue→taskBill 列表流 |
| **`.full.archimate`** | 变迁后拓扑：taskFE / APISIX / taskBill / billing_profit_sharing |
