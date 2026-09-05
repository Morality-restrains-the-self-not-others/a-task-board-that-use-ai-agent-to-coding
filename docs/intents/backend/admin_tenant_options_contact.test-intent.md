# 测试意图：管理端租户下拉返回创建者邮箱与手机号

## 测试目标

证明 tenant-options 返回创建者 `email`/`phone`，搜索可按联系方式命中，鉴权与 fail-open 成立。

## 测试分层

| 层 | 位置 |
|----|------|
| Go 单测 | `taskAuth/src/auth_users_test.go` batch/details 含 phone |
| Go 单测 | `taskTenantService/src/admin_tenant_options_test.go` |

## 用例矩阵

| 场景 | 期望 |
|------|------|
| 创建者有邮箱+手机 | JSON 含对应字段 |
| 创建者无登录方式 | email/phone 为空串 |
| search=邮箱/手机 | 命中其公司 |
| search=公司 ID | 命中该公司 |
| 无网关头 | 401 |
| 无平台角色 | 403 |
| taskAuth 不可用 | 200 且公司仍在，联系字段空 |
| 重复请求 GET | 结果稳定（只读） |

## 数据与环境

MySQL 测试库 + 可替换 `fetchCreatorContactsFn` / `searchCreatorIDsFn` 接缝；batch/details 用真实 login_method 行。

## 通过标准

上述测例全绿。
