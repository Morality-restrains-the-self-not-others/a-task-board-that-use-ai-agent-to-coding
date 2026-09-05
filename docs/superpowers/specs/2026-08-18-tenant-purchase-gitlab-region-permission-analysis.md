# 角色权限分析 — 租户购买 GitLab 须按行选区

- **日期**: 2026-08-18
- **设计**: `docs/superpowers/specs/2026-08-18-tenant-purchase-gitlab-region-design.md`
- **结论**: 无新角色、无新 endpoint。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST `/api/tenant/{tid}/billing/orders/` | tenant_admin | Tenant | write | `requireTenantAdmin` + 路径 tenant | ✅ | 增加 slug 白名单 |
| GET `/api/billing/gitlab-regions/tenant_id/{tid}/` | 租户成员 | Tenant | read | 既有列表 | ✅ | 仅 `is_active` |
| OrderCreate / OrderDetail | 同上 | Tenant | UI | 路由守卫 | ✅ | 区域为展示字段 |
| GitLab 设置页购买链接 | 租户管理员 | Tenant | navigate | 真实 `<a href>` | ✅ | 链到本租户 `orders/create/`，禁止 `@click.prevent` |

不引入跨租户下单。`region` 必须是已启用登记区。
