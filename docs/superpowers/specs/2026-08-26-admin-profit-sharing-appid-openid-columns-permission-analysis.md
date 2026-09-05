# 角色权限分析 — 管理端分账表 AppID / OpenID

- **日期**: 2026-08-26
- **设计**: `docs/superpowers/specs/2026-08-26-admin-profit-sharing-appid-openid-columns-design.md`
- **判定**: 绿灯 ✅

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/system-admin/profit-sharing/` 增 `app_id`/`openid` | 平台员工 | System | read | 网关 verified + `IsPlatformStaff` | ✅ | 仅 staff 可见；非 staff 403 不变 |
| FE 微信分账 Tab / 待分账面板 | 平台员工 | System | read | system-admin 壳 | ✅ | 壳外无入口 |
| GET `/api/billing/profit-sharing/referrer-orders/` | 推荐人本人 | Resource | read | X-User-Id 本人 | ✅ 不改 | **禁止**下发 openid |
| GET `/api/system-admin/orders/{id}/` `profit_sharing[]` | 平台员工 | System | read | IsPlatformStaff | ✅ 本轮不下发 | 范围自律 |

## 安全审查

- [x] **PII**: openid 仅平台员工列表；推荐人自助与展示名 enrich 仍禁止
- [x] **日志**: 不新增含 openid 的 slog 字段
- [x] **IDOR / 跨租户**: 无新写路径；过滤条件不变
- [x] **权限提升**: 无

## 权限测试

| 场景 | 角色 | 操作 | 预期 |
|------|------|------|------|
| staff 列表 | 平台员工 | GET | 200 且含 `app_id`/`openid` |
| 非 staff | 普通用户 | GET | 403 |
| 推荐人自助列表 | 推荐人 | GET referrer-orders | 响应不含 openid |
