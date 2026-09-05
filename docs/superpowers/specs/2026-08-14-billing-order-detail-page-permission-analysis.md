# 订单详情页 — 角色权限分析

输入：`docs/superpowers/specs/2026-08-14-billing-order-detail-page-design.md`

## 结论

无新 API、无新角色。详情页是 `billing.orders` 的子路由，复用 page `billing.orders` + region `billing.orders.main`。不新增 `auth_resource_group` 种子。

## 改动点

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| FE `GET .../orders/{id}/` | billing:view | Tenant | read | 网关鉴权 + 既有 GET | ⚠️ GET handler 未校验 path tenant 与订单归属（存量 IDOR） | 本次不改 API；记 OPT |
| FE `POST .../orders/` | tenant admin / billing:manage | Tenant | write | `requireTenantAdmin` | ✅ | — |
| FE `POST .../pay/` | 同上 | Tenant | write | 既有 pay handler | ✅ | — |
| 路由 `/billing/orders/:orderId/` | 同订单列表 | Tenant | read | 侧栏 `canSeeMenuKey('billing.orders')` | ✅ 无新菜单 | 不登记新 page |
| 列表「订单号」链接 | 同上 | Tenant | read | 真实 `<a href>` | ✅ | 禁止 `@click.prevent`+push |

## 建模

不引入新角色/权限码。
