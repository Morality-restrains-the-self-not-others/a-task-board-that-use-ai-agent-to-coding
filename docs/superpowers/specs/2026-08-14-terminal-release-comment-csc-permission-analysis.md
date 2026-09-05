# 角色权限分析：终态释放对齐评论级 CSC

- 日期：2026-08-14
- 设计：`docs/superpowers/specs/2026-08-14-terminal-release-comment-csc-design.md`
- 状态：已完成（goal-mode 自动推进）

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| PATCH todos（既有） | workspace 可写成员 | Task | write | hasWorkspaceAccess + canMutateTask | ✅ | 事件仅写库成功后发 |
| GET list-by-task | taskEvents 服务账号 | Tenant+Task CSC | read | X-Internal-Secret | ✅ | 校验 query tenant/workspace/task 与事件一致 |
| release-servers Dispatch | taskEvents | Comment CSC | write 副作用 | 服务凭据 + payload 边界 | ✅ | 禁止只凭 task_id 跨租户列评论 |
| CLOUD_SERVER_STOPPED | taskEvents | Instance | delete | 既有 stopped 链路 | ✅ | 信封自带 instance/region/comment |

## 结论

- 不新增公网 API、不新增角色。
- list-by-task 仅 internal；缺密钥 403。
- 无 IDOR：列表必须带 tenant_id + task_id；consumer 用事件内 company/tenant 查询。
