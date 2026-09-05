# 测试意图：管理端租户目录分页列表

## 测试目标

证明 tenants 列表鉴权、分页、空结果、联系方式 fail-open。

## 测试分层

| 层 | 位置 |
|----|------|
| Go 单测 | `taskTenantService/src/admin_tenants_list_test.go` |

## 用例矩阵

| 场景 | 期望 |
|------|------|
| 无网关头 | 401 |
| 无平台角色 | 403 |
| seed 一公司 | 200，items 含 c1，total≥1 |
| limit=1&offset=1（两公司） | 第二页另一 id |
| Auth 不可用 | 200，email/phone 空串 |
| 重复 GET | 结果稳定 |

## 通过标准

上述测例全绿。
