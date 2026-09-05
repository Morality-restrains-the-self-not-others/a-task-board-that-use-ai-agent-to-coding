# NFR 澄清：管理端赠送显式数量

支撑等级：L2。配额/会员写路径幂等 ≥ L3。无新架构。

## 路径分片键审视

| 路径 | 是否携带分片 ID | 该 ID 是否合适 | 可伸缩性 | 动作 |
|------|-----------------|----------------|----------|------|
| POST `/api/tenant/{tenant_id}/billing/accounts/admin_grant_points/` | 是 `tenant_id` | 是，租户账本分片键 | L1 | 维持路径带 tenant_id |
| FE `/system-admin/grant-points` | 否 | n/a | L0 | 超管 SPA；升级：管理赠送 QPS>5 |

Hard Gate：通过。

## 幂等性审视

| 路径 | 副作用 | 等级 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------|------------|--------------|--------|----------|
| POST admin_grant_points | 写 grant/订单/会员 | L3 | 双击、超时重试 | 一次确认赠送批次 | 点击生成的 `Idempotency-Key`（header+body 同值） | 同键返回已有 transaction |
| 数量校验/预览 | 无 | L0 | — | — | — | 纯 UI |

前端：`createClickGuard` 同步门闩 + debounce；禁止每次重试换新 UUID。

Hard Gate：通过。
