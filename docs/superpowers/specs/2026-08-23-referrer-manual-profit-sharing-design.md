# 推荐人冻结/可分账订单与手动分账

日期：2026-08-23  
状态：accepted（/goal 跳过 Step 1 用户闸门）

## 问题

推荐页「月份消费收益」展示的是计提汇总，推荐人无法看到订单完成 15 天内的冻结单、15–30 天可分账单，也无法在微信约 30 天窗口内主动发起分账。原先 `settle_after` 到期后小时扫描会自动分账，按钮几乎没有可点窗口。

## 决策

1. 冻结时钟以资源订单 `paid_at`（交易完成）为准：**15 天内冻结**，**15–30 天可分账**，**≥30 天过期无法分账**。
2. 推荐人在本人推荐页列表点击「分账」→ `POST /api/billing/profit-sharing/referrer-orders/{id}/share/`。
3. 小时扫描**不再**对 `settle_after<=now` 的 pending 自动执行；**保留 25–30 天系统兜底**（用户未点时的最后一次申请）。
4. 新行 `settle_after = paid_at + 15d`（与冻结对齐）。幂等键仍是 `out_profit_sharing_no`；前端同步门闩 + `Idempotency-Key`。
5. 列表与响应**禁止**返回 openid / 微信分账单号。仅推荐人本人（`referrer_user_id == X-User-Id`）。

## 非目标

- 不改买家订单详情（仍不向买家泄漏分账）。
- 不新建服务、不新建 Kafka 领域事件（与现网分账写路径一致：更新 `billing_profit_sharing.status` + 出站微信 CreateOrder）。
- 不把 25 天兜底改成 15 天自动打款。

## 展示态

| display_status | 条件 | 按钮 |
|---|---|---|
| frozen | `pending` 且 `now < paid_at+15d` | 无 |
| failed | `status=failed` 且（无 `paid_at` 或 `age < 15d`） | 无；**不得**显示冻结 |
| shareable | `pending`/`failed` 且 `15d ≤ age < 30d` | 分账（失败可重试） |
| expired | 未成功且 `age ≥ 30d` | 无，文案「已过期无法分账」 |
| processing | status=processing | 无 |
| shared | status=finished | 无 |
