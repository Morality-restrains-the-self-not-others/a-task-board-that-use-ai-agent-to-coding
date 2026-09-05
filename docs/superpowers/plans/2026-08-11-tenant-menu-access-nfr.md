# Step 5 — NFR：访问管理

| 类别 | 级别 | 说明 |
|------|------|------|
| Security | L3 | 授权写路径仅管理员；API RequirePerm；前端隐藏非安全边界 |
| Performance | L2 | 角色列表/成员列表现有分页或全量可接受（租户级） |
| Reliability | L2 | 保存失败展示 traceId；角色创建成功但绑定失败需可感知错误 |
| Usability | L2 | 系统角色只读提示；无权限空态 |
| Observability | L2 | 复用既有 RoleChanged / TenantRoleChanged 日志 |

例外：纯配置 UI，无新 MQ 事件类型。
