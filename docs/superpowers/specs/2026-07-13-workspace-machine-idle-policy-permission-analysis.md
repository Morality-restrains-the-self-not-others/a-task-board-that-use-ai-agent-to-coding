# 权限分析：工作空间机器节点闲置策略

- 日期：2026-07-13
- 设计：`2026-07-13-workspace-machine-idle-policy-design.md`
- 结论：不新增角色；复用租户会话 + 工作空间归属校验

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET workspace-machine-policy | workspace_member | Workspace | read | gateway forward-auth + tenant/workspace path | ✅ | 与 indicators 同级 |
| PUT workspace-machine-policy | tenant_admin / workspace 写权限 | Workspace | write | 同 CPA 写路径 | ⚠️ | 与 start-vm 同鉴权；拒绝跨 tenant 写 |
| GET workspace-machine-summary | workspace_member | Workspace | read | 同 indicators | ✅ | — |
| start-vm enabled CPA 门禁 | 启动者 | Workspace+CPA | write | 会话已有 | ✅ 新增业务规则 | 403 + 明确 message |
| idle recycle intent | 系统（taskEvents） | Workspace | manage | 内部凭据 | ✅ | 仅 internal；不暴露公网 |
| 前端设置模态 | 能进 settings/task-panel 的用户 | Tenant settings | write | 侧栏菜单权限 | ✅ | 无新菜单 |

## IDOR 防护

- 所有 API 以 path 中 `tenant`/`workspace` 为范围；策略表 PK=`(company_id, workspace_id)`
- recycle 扫描按 policy 行迭代，stop 时二次校验 config.workspace_id

## 无新角色

本迭代不引入 reviewer/auditor 等角色。
