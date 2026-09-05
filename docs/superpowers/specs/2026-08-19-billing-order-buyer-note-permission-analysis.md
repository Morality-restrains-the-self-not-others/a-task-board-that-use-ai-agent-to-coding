# 角色权限 — 下单施工留言

- **Date:** 2026-08-19
- **Design:** `docs/superpowers/specs/2026-08-19-billing-order-buyer-note-design.md`

无新角色、无新权限码。

| 改动点 | 主体 | 资源 | 操作 | 现有检查 | 缺失 | 建议 |
|--------|------|------|------|----------|------|------|
| POST orders + `buyer_note` | 租户管理员 | Tenant 订单 | write | `requireTenantAdmin` | 无 | 非空留言仅当 items 含人工履约 SKU |
| GET order 含 `buyer_note` | 本租户成员 / 平台员工 | 本租户订单 | read | 租户路径 + IDOR 404 | 无 | 他租户 order_id → 404，不泄露留言 |
| 超管展开读 GET tenant orders | IsPlatformStaff | 任意租户订单 | read | 网关 staff + 租户路径 | 无 | 不新增超管专用字段接口 |
| 列表 GET | 同上 | 订单摘要 | read | 既有 | 无 | 列表可不回传正文，详情必回传 |

条件：`buyer_note` 仅创建时可写，后续不可 PATCH（一期不做修改接口，避免施工中途被改写）。
