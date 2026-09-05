# NFR：订单详情退款按钮

- **Date:** 2026-08-22
- **Level:** 资金路径沿用既有退款 L3；本增量 UI + 重放语义

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 动作 |
|------|---------|------|------|
| GET `/api/tenant/{tenant_id}/billing/orders/{order_id}/` | tenant_id + order_id | 合适（订单属租户片） | 无 |
| POST `/api/tenant/{tenant_id}/billing/refund-applications/` | tenant_id；body.order_id | 合适 | 无新路径；L0 升级触发：跨租户退款目录 |
| SPA `/tenant/{tid}/billing/orders/{orderId}/` | tenant + order | 合适 | 无 |

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| 打开退款弹层 | 无 | — | — | L0 | 纯 UI |
| POST refund-applications | 创建 pending 申请 | 双击、超时重试 | 同一订单一条活跃申请；租户同时仅一条 pending | 前端点击 `Idempotency-Key`；服务端本单已有活跃申请则返回该申请 | 同键/同单重放不新建；他单在租户已有 pending 仍 409 |
| 超管批准（既有） | 原路退 + 收回剩余配额 | 审批重试 | application_id | 既有 provider progress | 不改 |

资金路径 ≥ L3：前端锁不替代「一单一条活跃申请」。禁止用 `user_id` 作消费键（本路径无新 Kafka 消费者）。
