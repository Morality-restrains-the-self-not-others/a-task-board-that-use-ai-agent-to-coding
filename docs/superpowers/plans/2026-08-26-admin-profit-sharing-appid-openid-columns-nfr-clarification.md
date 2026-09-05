# NFR 澄清 — 管理端分账表 AppID / OpenID

- **日期**: 2026-08-26
- **价值流**: `docs/superpowers/plans/2026-08-26-admin-profit-sharing-appid-openid-columns-value-stream.md`
- **默认等级**: L2；安全 L3（PII 仅 staff）

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 等级 | 结论 / 动作 |
|------|---------|------|------|-------------|
| GET `/api/system-admin/profit-sharing/` | 无租户 ID | 缺租户键 | **L0** | 既有超管跨租户列表；LEFT JOIN 1:1 不改分页。升级触发：单页仍 ≤50 |
| FE `/system-admin/users/` 抽屉 Tab | 无 | 缺键 | **L0** | 管理后台单操作者 |
| GET `/api/billing/profit-sharing/referrer-orders/` | user_id（本人） | 合适 | **L0** | 本增量不改键 |

无分片 ID 标 L0 理由：平台员工低频只读 + 已分页。`referrer_user_id` 是过滤键而非租户分片键。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| GET 列表增字段 | 无 | 刷新 | — | — | **L0** |
| 分账 POST share | 本增量不改 | — | — | 既有 Idempotency-Key | — |

无 Kafka / Webhook / timer 新路径。

## 类别定级

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L0 | 见上 |
| 数据一致性 | L1 | 展示台账快照，不现查微信身份 |
| 安全 | L3 | staff only；推荐人自助禁 openid |
| 可用性 | L2 | JOIN 失败即列表失败（与现网一致） |
| 性能 | L2 | 主键 JOIN，无 N+1 |
| 可观测性 | L2 | 不打 openid 明文 |
| 幂等 | L0 | 无写 |

## 质量场景

1. 刺激：staff 打开微信 Tab 且有登记接收方。响应：表格可见 AppID 与 OpenID。
2. 刺激：非 staff GET 列表。响应：403。
3. 刺激：推荐人打开自助分账页。响应：文案与 JSON 均无 openid。

## 领域模型影响

不新增聚合。查询 DTO 增加两个只读字段。无纠正分片键。
