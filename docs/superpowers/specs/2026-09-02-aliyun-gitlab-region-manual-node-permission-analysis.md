# 角色权限分析 — 阿里云 GitLab 区域人工建节点

- **日期**: 2026-09-02
- **设计**: `docs/superpowers/specs/2026-09-02-aliyun-gitlab-region-manual-node-design.md`

无新角色。沿用租户 VIP1 购买 + system-admin 区域 CRUD。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/billing/gitlab-regions/tenant_id/{tid}/` | 租户成员（VIP1 购买页） | Tenant | read | 网关会话 + tenant 路径 | ✅ | 列表不返回 token（既有清空） |
| POST `/api/tenant/{tid}/billing/orders/` | tenant 计费权限 | Tenant | write | 既有 createOrder + 会员门槛 | ✅ | region 须 is_active；pending_node 允许 |
| 支付 markOrderPaid | 支付回调/内部 | Tenant | write | 订单归属 tenant_id | ✅ | 事件 key=order_id 非 tenant_id |
| PUT `/api/system-admin/gitlab-regions/{slug}/` infra_status | system-admin | System | write | 网关 system-admin | ✅ | 禁止租户改 infra_status |
| POST admin provision | system-admin / internal secret | Tenant+Region | write | requireTenantAdmin 或 internal | ✅ | pending_node 须 409，禁止对空实例开通 |
| GET 区域列表（含 pending_node） | 租户 | Tenant | read | access_mode 过滤 tester | ✅ | Aliyun seed 为 release |

IDOR：开通/配额仍按 `(tenant_id, region)`；管理员 PUT 按 slug 全局配置（system-admin 预期）。

不新增 Python 接口。不新增租户可写的 infra_status。
