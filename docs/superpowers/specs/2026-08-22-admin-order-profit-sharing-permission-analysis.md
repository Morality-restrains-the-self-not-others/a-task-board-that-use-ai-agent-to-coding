# 角色权限分析：管理员订单分账只读

- **日期**: 2026-08-22
- **设计**: `docs/superpowers/specs/2026-08-22-admin-order-profit-sharing-design.md`

## 权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/system-admin/orders/{order_id}/` | 平台员工 | System | read | 与列表相同：网关头 + IsPlatformStaff | ✅ 充分 | 拒绝时 WARN 日志（无 PII） |
| GET `/api/tenant/{tid}/billing/orders/{id}/` | 租户成员 | Tenant | read | 路径 tenant + 订单归属 IDOR | ✅ 充分 | **禁止**附加 profit_sharing |
| `OrderExpandDetail` profitSharing prop | 调用方决定 | UI | read | 管理员传入数组；租户不传 | ✅ | 默认 null 不渲染 |
| `useAdminOrderRowExpand` | 仅管理页 | System | read | 改打 admin GET | ✅ | 勿回退租户 GET |

## 角色建模

不新增角色。沿用 `super_admin` / `employee`（`authz.IsPlatformStaff`）。

## IDOR

管理员详情按订单主键跨租户读取，仅 staff。租户详情仍须 `tenant_id` 与订单归属一致（既有）。

## 数据最小化

响应不含 `referrer_openid`、微信分账单号。金额以分为整数 + 元字符串。
