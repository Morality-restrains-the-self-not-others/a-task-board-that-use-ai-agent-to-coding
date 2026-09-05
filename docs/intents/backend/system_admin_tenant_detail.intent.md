# 功能意图：系统管理租户详情只读 API

- **日期**: 2026-08-25
- **状态**: 已交付

## 背景与目标

平台员工按租户 ID 读取公司头、剩余配额、工作空间列表；订单列表支持 `tenant_id` 过滤。

## 范围

- `GET /api/system-admin/accounts/admin/tenants/{id}/`
- `GET /api/system-admin/tenant-quotas/tenant_id/{id}/`
- `GET /api/system-admin/tenant-workspaces/tenant_id/{id}/`
- `GET /api/system-admin/orders/?tenant_id=`

## 约束

- 鉴权：网关已验证 + `IsPlatformStaff`
- 数据所有权：各 owner 只读本服务表
- 无 MQ 事件（纯查询）

## 验收标准

1. 非 staff → 403；缺网关验证 → 401
2. 未知公司 header → 404
3. quotas 响应字段与租户 quotas 同构
4. workspaces 返回该 `company_id` 全部行（无 mine）
5. orders 带合法 `tenant_id` 只返回该租户

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 例外理由 |
|----------|--------|----------|
| 平台员工查询租户详情数据 | — | 纯 GET |

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-25 | 初稿 |
