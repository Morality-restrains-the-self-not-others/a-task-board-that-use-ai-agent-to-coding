# 测试意图：超管用户列表 tenant_companies 回填

## 测试目标

batch-get 聚合与列表 handler 契约。

## 测试分层

| 层 | 位置 |
|----|------|
| 聚合纯函数 | `taskAuth/src/tenant_companies_client_test.go` |
| HTTP client | 同上 |
| 列表集成 | `taskAuth/src/handlers_system_admin_users_test.go` |

## 用例矩阵

| 场景 | 期望 |
|------|------|
| 活跃+停用混合 | 只保留活跃 |
| 空名 | name 回退 company_id |
| 多公司 | 按名称排序、按 company_id 去重 |
| 空 userIDs / 下游宕机 | 空 map，不 panic |
| 列表 mock 租户服务 | 目标用户带对应 tenant_companies |

## 通过标准

上述测例全绿。
