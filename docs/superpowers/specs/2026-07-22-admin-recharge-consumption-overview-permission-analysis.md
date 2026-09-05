# 角色权限：充值消费情况总览

| 端点 | 角色 | 权限 |
|------|------|------|
| Django `GET /api/system-admin/user-recharge-consumption/` | superuser | 读 |
| taskBill internal admin 汇总 | internal secret | 服务间 |
| 引荐页文案 | 登录用户本人 | 读 |

非超管访问 Django → 403。无写路径。
