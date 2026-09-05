# 权限分析：租户订单列表按交易单号 / 商户单号查询

- **日期**: 2026-08-29
- **设计**: `docs/superpowers/specs/2026-08-29-tenant-order-trade-no-query-design.md`

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/tenant/{tid}/billing/orders/?order_number=` | 租户成员（`billing:view` 或 `billing:manage`） | Tenant | read | 网关 forward-auth + 租户 PDP（与无 query 的列表相同）；SQL `tenant_id = ?` | ✅ 充分 | 保持；禁止去掉 tenant 约束 |
| 前端 `/tenant/:tenant/billing/orders/` 查询框 | 同上 | Tenant | read | 侧栏 `billing.orders` 已 `anyOfPerms: billing:view, billing:manage` | ✅ 充分 | 无新权限码 |
| 管理端 `GET /api/system-admin/orders/?order_number=` | 平台员工 | System | read | `IsPlatformStaff` | ✅ 不改 | 本增量不扩大超管面 |

## 越权场景

| 场景 | 期望 | 覆盖 |
|------|------|------|
| 用他租户订单的微信交易单号在本租户列表查询 | 200 + `orders=[]` `total=0`，不 404 泄露存在性 | 已有 `TestHandleListOrdersTradeNoDoesNotLeakOtherTenant`；本增量补交易单号/商户单号各一条 |
| 未登录 | 401（网关） | 既有网关行为 |
| 无 `billing:view` | 403 / 不可达该路由 | 既有 PDP + 侧栏 |

## 角色建模

不新增角色或权限码。

## IDOR

查询值不是路径资源 ID；过滤键始终带 URL 中的 `tid`。不得用客户端传入的 `tenant_id` 覆盖路径租户。
