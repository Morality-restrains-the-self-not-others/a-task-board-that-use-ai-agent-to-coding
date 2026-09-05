# ADR-0030: 订单分账明细仅管理员可见

- **Status:** accepted
- **Date:** 2026-08-22
- **Author:** cursor
- **Deciders:** goal-mode auto-pipeline

---

## Context

资源订单支付后，taskBill 将推荐分账写入 `billing_profit_sharing`（接收方用户 ID、佣金分、状态）。管理员在订单记录页展开行项时需要看到「分账给谁、分了多少」。租户订单列表/详情与管理员展开曾共用同一组件，且管理员展开调用租户订单 GET。分账属于资金分配信息，进入租户 API 即构成可探测泄漏。

## Decision

We will treat order-level profit-sharing as **platform-staff-only**:

1. Expose `profit_sharing[]` only on `GET /api/system-admin/orders/{order_id}/` (IsPlatformStaff).
2. Tenant `GET /api/tenant/{tenant_id}/billing/orders/{order_id}/` **omits** the `profit_sharing` key entirely.
3. Admin UI expand uses the admin detail endpoint; tenant UI never passes the profit-sharing prop into the shared expand component.
4. Response never includes WeChat `openid` or other payment-account identifiers.

## Alternatives Considered

### Alternative 1: Add field to tenant GET, hide in Vue

- **Pros:** 少一个 endpoint
- **Cons:** 任何持有租户会话的客户端都能读到分账
- **Why rejected:** 资金分配对租户保密是需求硬约束

### Alternative 2: Query param `include=profit_sharing` on tenant GET

- **Pros:** 复用 loadOrder
- **Cons:** 租户可自行加参；授权与资源路径不一致
- **Why rejected:** 权限应绑在管理员资源，而不是可选开关

## Consequences

### Positive

- 租户 API 契约不变（无新键）
- 管理端有明确详情资源，后续可加更多仅超管字段
- 与现有 `/api/system-admin/orders/*` 网关前缀一致

### Negative / Trade-offs

- 管理员展开从租户 GET 换成管理员 GET，测试夹具 URL 需更新
- 一单若从未 `markOrderForProfitSharing`，管理端显示空列表而非推断推荐边

### Follow-up

- 历史缺记录回填不在本 ADR 范围
