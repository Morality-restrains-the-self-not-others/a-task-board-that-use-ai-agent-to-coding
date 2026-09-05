# 角色权限分析 — 按交易单号查询订单

- **日期**: 2026-08-21
- **设计**: `docs/superpowers/specs/2026-08-21-system-admin-order-trade-no-query-design.md`

## 权限影响

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/system-admin/orders/?order_number=` | 平台员工 | System | read | `X-Gateway-Auth-Verified` + `authz.IsPlatformStaff` | ✅ 充分 | 沿用列表鉴权；精确匹配不扩大匿名面 |
| GET `/api/internal/taskbill/admin/orders/?order_number=` | 内部调用方 | System | read | `requireInternalSecret` | ✅ 充分 | 与公开管理端同一 `doAdminListOrders` |
| GET `/api/tenant/{tid}/billing/orders/?order_number=` | 租户管理员 | Tenant | read | `parseTenantID` + 既有租户访问 | ✅ 充分 | WHERE 必须含 `tenant_id`；禁止跨租户 |
| FE `/system-admin/order-records/` | 平台员工 | System | read | 既有 system-admin 路由守卫 | ✅ 充分 | 查询按钮只读，无新写权限 |

## 风险与缓解

- **IDOR**：租户列表若只按 `order_number` 不带 `tenant_id` 会泄露他租户订单 → SQL 强制 `tenant_id=?`。
- **枚举**：精确匹配 + staff 门禁；未命中空列表不区分「不存在 / 无权限」。管理端本就可列全量订单。
- **无新角色**。
