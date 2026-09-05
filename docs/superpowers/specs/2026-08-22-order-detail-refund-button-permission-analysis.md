# 角色权限分析：订单详情退款按钮

- **Date:** 2026-08-22
- **Design:** `2026-08-22-order-detail-refund-button-design.md`

无新角色、无新权限码、无新 HTTP 路径。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET 订单详情（含 consumption） | 租户成员 `billing:view` | Tenant / Order | read | 既有 tenant 成员 + 订单 tenant_id | ✅ | — |
| 详情页「申请退款」按钮 | 可见性按 `refund_enabled` + 订单资格；提交须管理员 | Tenant / Order | write | POST `authz.RequirePerm(billing:manage)` | ✅ | 非管理员点提交仍 403，与列表一致 |
| POST refund-applications | 租户管理员 | Tenant | write | `requireTenantAdminForRefund` | ✅ | 重放只返回本 tenant 本单申请，防 IDOR |
| 超管审批 | superuser | System | write | 既有 system-admin | ✅ 不改 | — |

IDOR：重放查询必须 `tenant_id + order_id` 限定，禁止只按幂等头跨租户返回。
