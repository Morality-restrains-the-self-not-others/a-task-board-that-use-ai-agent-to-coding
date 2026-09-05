# 角色权限分析 — GitLab 磁盘手动发货开通

- **日期**: 2026-09-03
- **设计**: `docs/superpowers/specs/2026-09-03-gitlab-disk-manual-admin-fulfillment-design.md`

无新角色。沿用租户 VIP1 购买 + system-admin 开通实施。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET 租户 GitLab 配额视图 | 租户成员 | Tenant+Region | read | 网关会话 + tenant 路径 | ✅ | GET **不得**再 ensure 组 |
| POST 资源订单 / 支付回调 | 租户 / 支付 | Tenant | write | 既有 createOrder + markOrderPaid 分片键 | ✅ | 只写 pending_admin 额度 |
| POST `/api/tenant/{tid}/billing/gitlab-resources/provision/` | system-admin 或 internal secret | Tenant+Region | write | `requireTenantAdmin` 或 internal | ✅ | 购买路径唯一建组入口；租户会话不得调 |
| GET `/api/system-admin/gitlab-resources/pending-fulfillment/` | system-admin | System | read | 网关 system-admin | 🆕 | 禁止租户枚举全平台待开通 |
| 管理端赠送 ensure 组 | system-admin | Tenant+Region | write | 既有 grant | ✅ | 赠送视为管理员已发货 |

IDOR：provision 与配额仍按路径 `tenant_id` + body `region`；待开通列表仅超管可见，返回的 tenant_id 用于填表，不能给租户。

事件 key：`GitlabDiskFulfillmentQueued` 用 `order_id`；`GitlabTenantResourceProvisioned` 用 `tenant_id:region`（与开通幂等边界同粒度，禁止只用 tenant_id）。

不新增 Python 接口。
