# 角色权限分析：微信关联账号查单

## 结论

沿用管理端订单列表既有员工门禁 + taskAuth 内部密钥。不新增角色。

## 端点

| 端点 | 认证 | 授权 | 数据范围 | 拒绝 |
|------|------|------|----------|------|
| GET `/api/system-admin/orders/?wechat_account=` | `X-Gateway-Auth-Verified=1` | `authz.IsPlatformStaff` | 全租户订单中命中行 | 401 / 403 |
| GET `/api/internal/users/wechat-linked-account/` | 内部密钥 | 仅内部 | 仅返回 user_id | 403；浏览器不可达（网关不暴露） |

## 前端

仅系统管理订单页渲染 Tab；非员工进不了该路由（既有路由守卫）。查询失败不得在 UI 打印 openid。

## 审计

结构化日志：`wechat_linked_account_lookup` / `admin_order_wechat_account_query`，含 `trace_id`、`query_len`、`user_count`/`total`，不含查询原文与 openid。
