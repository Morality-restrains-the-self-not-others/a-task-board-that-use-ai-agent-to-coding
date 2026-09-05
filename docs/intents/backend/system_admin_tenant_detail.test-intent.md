# 测试意图：系统管理租户详情只读 API

## 测试目标

证明平台员工可按租户 ID 读取头/配额/工作空间，订单可按 tenant_id 过滤；拒绝非员工。

## 测试分层

| 层 | 位置 |
|----|------|
| taskTenantService | `admin_tenants_get_test.go` |
| taskBill quotas | `admin_tenant_quotas_test.go` |
| taskBill orders | 扩展 `handlers_admin_list_orders` 测例 |
| taskProjectService | `admin_tenant_workspaces_test.go` |

## 用例矩阵

| 场景 | 期望 |
|------|------|
| GET tenant 无 staff | 403 |
| GET tenant 未知 id | 404 |
| GET tenant 存在 | 200 + name |
| GET quotas staff | 200 + task_post_quota 键 |
| GET quotas 非 staff | 403 |
| GET workspaces staff | 200 items |
| GET workspaces 非 staff | 403 |
| GET orders ?tenant_id= | 仅该租户订单 |
| GET orders 非法 tenant_id | 400 |

## 通过标准

上述测例全绿。
