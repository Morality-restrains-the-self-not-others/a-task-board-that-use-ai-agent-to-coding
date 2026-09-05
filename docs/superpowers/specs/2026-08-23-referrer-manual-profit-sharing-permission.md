# 权限分析：推荐人手动分账

## 角色

| 角色 | GET 列表 | POST 分账 |
|------|----------|-----------|
| 匿名 | 401 | 401 |
| 已登录推荐人（行上 referrer） | 200 本行 | shareable 时 200 |
| 已登录但非该行推荐人 | 列表不含该行；POST 404 | 404 |
| 平台员工 | 不走本 API（仍用 system-admin 队列） | 不走本 API |

## 规则

- 网关 token + `X-Gateway-Auth-Verified=1` + `X-User-Id`。
- **禁止** query 覆盖 referrer_user_id。
- 不授予租户管理员查看他人分账的权限。
