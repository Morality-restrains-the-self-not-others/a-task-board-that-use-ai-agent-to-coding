# 价值流 — 超管在推荐绩效抽屉核对微信分账

- **日期**: 2026-08-23
- **增量**: 单切片，只读运营核对

## Related Value Streams

- `2026-08-22` 管理员待分账队列：全局 `GET /api/system-admin/profit-sharing/`。本增量**扩展**同接口 `referrer_user_id`，不改默认队列。
- 推荐绩效抽屉（支付明细 / 佣金）：taskReferral `referral-performance`。本增量分账数据走 taskBill，不经 Referral 库。

## 当前价值流（增量）

1. 超管打开 `/system-admin/users/` → 推荐绩效抽屉。
2. 默认 Tab「支付明细」（既有）。
3. 切「微信分账」→ `GET /api/system-admin/profit-sharing/?referrer_user_id={uid}&status=all`。
4. taskBill：推荐人 → 被推荐人边 → 买家订单 → INNER JOIN `billing_profit_sharing`。
5. 表格展示被推荐人、订单号、金额、本地状态；接收方登记摘要不含 openid。
6. 「同步微信状态」→ `POST .../refresh-wechat/` 对当前页逐笔 QueryOrder；不回写 DB。

## 测试点

| ID | 步骤 | 断言 |
|----|------|------|
| TP-RPS-1 | 推荐人 A 仅 U1 订单有台账 | 列表仅 U1，含 `referred_user_id` |
| TP-RPS-2 | U2 已支付无台账行 | 不出现 |
| TP-RPS-3 | 不传 referrer_user_id | 全局队列行为不变 |
| TP-RPS-4 | 响应 | 无 openid |
| TP-RPS-5 | 非 staff | 403 |
| TP-RPS-6 | refresh 他人 id | 400 |
| TP-RPS-7 | QueryOrder FINISHED | 响应 wechat_state，DB 不变 |
| TP-RPS-8 | 抽屉默认 | 支付明细；点 Tab 才请求分账 |
| TP-RPS-9 | 同步连点 | guard 第二次不发 |

无新 MQ 事件（只读 + 出站只读）。
