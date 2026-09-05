# 权限分析：task-detail-machine-owner-hint

- **日期**: 2026-07-15
- **关联设计**: `2026-07-15-task-detail-machine-owner-hint-design.md`
- **结论**: 无新增公网端点；扩展字段沿用既有 `server-runtime-status` 鉴权与租户/工作区范围。

## 改动点权限矩阵

| 改动 | 主体 | 资源 | 操作 | 范围 | 条件 | 风险 | 缓解 |
|------|------|------|------|------|------|------|------|
| `server-runtime-status` 响应增加绑定任务列表 | 已登录租户成员（与现接口相同） | Workspace CSC | read | 同 tenant+workspace | 仅能查本任务 status；同实例兄弟 task_id 属同 workspace | 泄露同工作区其他任务 ID | 可接受：工作区内任务本就可互相发现；不返回跨 workspace；不含密钥 |
| 前端镜像区展示 / 链接 | 同上 | Task detail UI | read | 同 workspace | 仅容器运行时展示 | 无 | 链接仍走现有路由鉴权 |

## 不引入的权限面

- 不开放 internal `instance-bindings` 到公网
- 不新增写接口
- 不跨租户查询

## 审计结论

✅ 可进入实现；无需额外 RBAC 角色。
