# 角色权限 — 管理端赠送页修改 VIP 等级

- **日期**: 2026-08-18
- **设计**: `docs/superpowers/specs/2026-08-18-admin-grant-membership-tier-design.md`

无新角色。VIP 变更与赠送同一网关：system-admin + Billing 代理，禁止租户自调。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST `.../billing/accounts/admin_grant_points/` + `membership_tier` | 系统管理员 | System → 指定 Tenant 会员 | write | 网关 system-admin + 既有赠送代理 | ✅ 充分 | 白名单 `normal`/`vip1`；租户会话不可达此管理页 |
| GET `.../billing/membership/`（选中租户后展示） | 系统管理员（URL 中 tid） | Tenant 会员只读 | read | 与赠送同一代理 | ✅ 充分 | 仅展示，不在此 GET 写等级 |
| POST 内部 `admin-grant-resources` | 内部服务 | Tenant 配额 | write | Internal Secret | ✅ 充分 | **不解析** `membership_tier`，避免礼包/佣金改 VIP |
| 租户 GET/自助 membership | 租户成员 | 本租户 | read | 既有 | ✅ 充分 | 不新增租户写接口 |

## 建模

不新增角色。`super_admin` 继续独占赠送页。

## IDOR

`tid` 来自 URL；系统管理员可对任意租户写 VIP。租户不能把自身 URL 改成管理赠送路径（前端路由 + 网关角色）。
