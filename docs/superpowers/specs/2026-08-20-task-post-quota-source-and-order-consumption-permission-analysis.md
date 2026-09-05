# 角色权限分析 — 任务帖配额来源与订单消耗归属

- **Date:** 2026-08-20
- **Design:** `docs/superpowers/specs/2026-08-20-task-post-quota-source-and-order-consumption-design.md`

## 结论

不新增角色/权限码。读写均落在既有租户账单接口上；须继续强制路径 `tenant_id` 与会话租户一致，防止 IDOR。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/billing/quotas/` 增字段 | 租户成员 `billing:view` | Tenant | read | 路径 tenant + 网关/PDP | ✅ | 新字段不扩大可见范围 |
| GET `/billing/orders/{id}/` 增 consumption | 同上 | Tenant | read | loadOrder(tenant, id) | ✅ | events 仅该订单 related_order_id |
| markOrderPaid 写 purchase grant | 支付回调/租户支付 | Tenant | write | 订单归属 tenant | ✅ | grant.tenant_id=订单 tenant |
| consumeTaskPostQuotaTx | 内部创建/续存 | Tenant | write | internal + tenant | ✅ | FOR UPDATE 同租户批次 |
| ensureTaskPostPurchaseLots | 同租户 GET | Tenant | write(repair) | 仅该 tenant | ✅ | 禁止跨租户回填 |
| 退款清 remaining | 退款审批 | Tenant | write | 既有退款授权 | ✅ | 只清本 order_id 批次 |

## 建模

无新角色。`page:billing` / `page:billing.orders` 不变。

## IDOR

订单消耗 events 必须 `related_order_id=该订单` 且订单 `tenant_id` 已校验。禁止用 task_id 跨租户反查。
