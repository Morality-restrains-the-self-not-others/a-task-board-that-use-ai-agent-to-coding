# 订单电子发票 — 角色权限分析

设计：`docs/superpowers/specs/2026-08-23-order-invoice-wechat-fapiao-design.md`

无新角色。沿用 `billing:view` / `billing:manage` / `IsPlatformStaff`。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST .../orders/{oid}/invoice-applications/ | 租户管理员 billing:manage | Tenant/Order | write | RequirePerm + path tid + loadOrder(tid,oid) | ✅ | 跨租户 404 |
| GET .../orders/{oid}/ 的 invoices[] | billing:view 或订单可读角色 | Tenant/Order | read | 与订单 GET 相同 | ✅ | 不含买家手机号明文 |
| GET /api/system-admin/invoice-applications/ | 平台员工 | System | read | IsPlatformStaff + gateway verified | ✅ | 无 openid |
| POST .../invoice-applications/{id}/approve/ | 平台员工 | System | write | 同上 | ✅ | 资金/税务副作用，须 staff |
| POST .../reject/ | 平台员工 | System | write | 同上 | ✅ | |
| POST /api/billing/wechat/fapiao/notify/ | 微信支付 | System | write | notify.Handler 验签 | ✅ | auth_mode machine |
| 退款 approve 触发红冲重开 | 平台员工（既有退款） | Order | write | 既有 approve | ✅ | 不新开口 |
| GET 订单 `invoice_reverse_confirm` / invoices[] 72h 截止 | 与订单 GET 相同 | Tenant/Order | read | 同订单可读角色 | ✅ | 只读计算字段，无新写口；截止时间非 PII |

IDOR：申请与列表必须 `tenant_id` + `order_id` 双条件。买家税号/账号仅员工与本租户管理员可见，日志只打 invoice_id 指纹。
