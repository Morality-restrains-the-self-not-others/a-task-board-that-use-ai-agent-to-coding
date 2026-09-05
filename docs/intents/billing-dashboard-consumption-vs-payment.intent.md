# 账单首页：累计消耗 vs 累计支付

## 背景与目标

租户账单页 `/tenant/{id}/billing/` 同时展示「累计消耗」与「累计支付」。二者口径不同，且前端曾误读 `*_cents`（API 契约为 `*_points`），导致卡片恒为 `0.00`，用户无法理解差异。

## 范围与边界

- **累计消耗**：`billing_transaction.transaction_type = consumption` 金额合计（用量扣费、配额消耗、以及仍为 `paid` 的资源订单 `resource_purchase`）。**不含**已退款（`refunded`）、已取消（`cancelled`）订单对应的购买流水。
- **累计支付**：用户真实付钱入账，不含后台赠送 `admin_grant`。包括钱包充值（`user_recharge` / PayPal / 微信 / 管理员直充）以及资源订单实付（`consumption` + `resource_purchase`），并减去 `transaction_type=refund` 负金额。
- **不含（消耗）**：已退款/已取消订单的 `resource_purchase`。钱包充值退款不计入消耗（避免把消耗打成负数）。
- **不含（支付）**：后台赠送。退款通过负金额流水冲减，不要再排除已退订单购买以免双重扣减。
- 只读聚合 `GET /api/tenant/{tid}/billing/statistics/`；前端展示元（分/100）。

## 约束与风险

- 金额单位：`*_points` 与兼容别名 `*_cents` 均为人民币分。
- 不得把赠送计入累计支付。
- 卡片须用文案说明二者差异，避免用户以为应对齐。

## 验收标准

1. API 返回 `total_consumption_points` / `user_recharge_points`，并带同值 `*_cents` 别名。
2. 前端优先读 `*_points`，兼容 `*_cents`；页面按元展示。
3. 资源订单实付计入累计支付；`admin_grant` 不计入。
4. 「累计消耗」「累计支付」卡片各有口径说明；消耗文案标明不含已退款、已取消订单。
5. 已退款/已取消订单的 `resource_purchase` 不计入 `total_consumption_*` / `monthly_consumption_*`。

## 变更记录

- 2026-08-20：消耗口径排除 `refunded`/`cancelled` 资源订单购买；支付侧仍用 purchase+refund 净值。
- 2026-08-19：修复字段名错配；累计支付纳入资源订单实付；补齐卡片口径说明。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 查看账单累计消耗/支付 | — | — | — | — | 只读聚合查询，不改变系统事实 |
