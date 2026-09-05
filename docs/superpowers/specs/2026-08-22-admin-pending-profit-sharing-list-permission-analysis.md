# 角色权限分析 — 管理员待分账订单列表

- **日期**: 2026-08-22
- **设计**: `docs/superpowers/specs/2026-08-22-admin-pending-profit-sharing-list-design.md`

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/system-admin/profit-sharing/` | 平台员工 | System | read | 拟：网关 verified + `IsPlatformStaff` | 实现时必须对齐订单列表 | 禁止仅靠前端隐藏 |
| FE Tab `?tab=profit-sharing` | 平台员工 | System | read | 既有 system-admin 壳 | ✅ | 壳外无入口 |
| 租户任意路径 | 租户成员 | Tenant | — | 无此 API | ✅ | 不注册 tenant 前缀 |
| 响应字段 | 平台员工 | System | read | ADR-0030 | ✅ | 禁止 openid / 微信分账单号 |

## 角色建模

不新增角色。复用 `authz.IsPlatformStaff`（super_admin / employee）。

## IDOR / 越权

- 列表无按租户隔离（平台运营跨租户队列是需求）；非 staff 必须 403。
- 订单号深链仍落在既有管理员订单 Tab，不把租户带进本 API。

## 审计

只读 GET，不写审计表。资金写路径仍走既有分账 timer。
