# NFR 澄清 — 管理员待分账订单列表

- **日期**: 2026-08-22
- **价值流**: `docs/superpowers/plans/2026-08-22-admin-pending-profit-sharing-list-value-stream.md`
- **默认等级**: L2；资金**写**不在本增量；读路径安全 L3（资金分配仅 staff）

## 路径分片键审视

| 路径 | 分片 ID | 判定 | 可伸缩性等级 | 结论 / 动作 |
|------|---------|------|-------------|-------------|
| GET `/api/system-admin/profit-sharing/` | 无 | 缺键 | **L0** | 平台员工低频跨租户运维队列；`billing_profit_sharing` 年增量远低于百万。升级触发：表 > 100 万行或 P95 > 500ms 时强制 `tenant_id` query 并按租户分片检索 |
| FE `/system-admin/order-records/?tab=profit-sharing` | 无 | 缺键 | **L0** | 管理后台单页；人数极少。升级触发：同 API |
| 订单号深链 `/system-admin/order-records/?tenant_id=&order_id=` | `tenant_id`（query） | 合适作后续点查 | L0/L2 | 沿用既有订单 Tab；本增量不改 |

无分片 ID 标 L0 的理由：管理端只读目录 + 可预见窗口单实例可承受 + 写明升级触发。禁止跨分片默扫的写路径不在本增量。

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| GET profit-sharing 列表 | 无 | 刷新/Tab 切换 | — | — | **L0** 同一快照 |
| FE Tab 点击 | 无（只改 query） | 连点 | — | — | Anti-Replay-OK: 只读 Tab；不生成 Idempotency-Key |
| 刷新按钮 | 无 | 连点 | — | — | Anti-Replay-OK: 只读 GET |

无 HTTP 写、无 Kafka、无 Webhook、无 timer。分账执行仍是既有 `billing_profit_sharing_scan`（本增量不改）。禁止用 `tenant_id`/`user_id` 作消费键 — 本增量无消费者。

资金/配额路径默认 ≥ L3 适用于**写**；本增量仅为读队列。

## 类别定级

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L0 | 管理端全局队列；升级触发见上 |
| 数据一致性 | L0 | 只读已落库快照 |
| 安全 | L3 | staff 门禁；无 openid；租户不可见 |
| 可用性 | L2 | 空列表合法；错误带 trace_id |
| 性能 | L2 | `idx_profit_sharing_status_settle`；分页 ≤ 50 |
| 可观测性 | L2 | status + total；禁 PII/openid |

## 质量场景

1. 刺激：staff 打开待分账 Tab 且库中有 pending。响应：表格见该行。
2. 刺激：仅 finished。响应：默认 open 为空文案。
3. 刺激：member GET。响应：403。
4. 刺激：响应 JSON。响应：无 openid。

## 领域模型影响

新增只读投影 `ProfitSharingQueueItem`，不改变分账聚合写边界。无需纠正分片键。
