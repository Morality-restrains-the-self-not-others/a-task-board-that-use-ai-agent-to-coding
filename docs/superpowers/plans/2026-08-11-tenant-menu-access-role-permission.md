# Step 2 — Role-Permission：访问管理

## 新/改路径权限

| 路径 | 操作 | 所需权限 | 说明 |
|------|------|----------|------|
| `/tenant/:t/people/access/` | 查看/配置 UI | `member:manage` | 页内门禁 |
| `GET /api/auth/roles/company_id/{cid}/` | 列角色 | 租户成员 | 已有 |
| `POST /api/auth/roles/` | 建自定义角色 | `member:manage` | 已有 |
| `PUT /api/auth/roles/role_id/{rid}/` | 更新权限 | `member:manage` | 已有 |
| `PUT /api/tenant/member-role/...` | 绑成员角色 | `member:manage` | 已有 |
| `PUT /api/tenant/group-role/...` | 绑组角色 | `group:manage` | 已有；页内保存组时需同时有此码，否则提示 |

## 数据访问

- 不新增表；读写既有 auth_role / tenant_member_role / tenant_group_role。
- 禁止无权限用户枚举并修改他人角色（后端 RequirePerm 已保证）。

## 审计结论

无新增后端 endpoint；前端消费既有 API。风险：管理员可降级自己 → UI 警告但不阻断（与 v63 一致）。
