# 资源订单号全局唯一 — 角色权限分析

- **Date:** 2026-08-19
- **Design:** `docs/superpowers/specs/2026-08-19-billing-order-number-uniqueness-design.md`

无新角色、无新 endpoint、无新资源组。变更只影响 `order_number` 生成与展示。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `POST /api/tenant/{tid}/billing/orders/` 写入 order_number | 租户管理员 `billing:manage` | Tenant | write | 租户路径 + 账单权限 | ✅ | 号码全局唯一不改变授权边界 |
| `GET /api/tenant/{tid}/billing/orders/` 展示 order_number | `billing:view` | Tenant | read | `tid` 过滤 | ✅ | 列表不得漏出他租户订单（已有） |
| `GET /api/tenant/{tid}/billing/orders/{order_id}/` | `billing:view` | Tenant | read | tid + order 归属 | ✅ | URL 用 id 不靠 order_number；IDOR 测已有 |
| 系统管理员订单记录页 | superuser | System | read | 管理端鉴权 | ✅ | 全局唯一号码避免跨租户同号歧义 |
| `adminGrantResources` / backfill 生成 order_number | 系统管理员 | Tenant | write | 管理端 | ✅ | 与用户下单同一生成函数 |

**IDOR：** 详情与支付按 Snowflake `id` + `tenant_id`，不按展示号解析。禁止新增「仅 order_number、无 tenant」的公开查询接口。
