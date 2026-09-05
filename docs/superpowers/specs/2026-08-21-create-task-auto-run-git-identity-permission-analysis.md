# 角色权限分析：创建任务自动运行 Git 提交身份

- **日期**: 2026-08-21
- **设计**: `2026-08-21-create-task-auto-run-git-identity-design.md`

## 结论

无新角色、无新 endpoint。沿用创建/更新任务的工作空间写权限与 Git 身份「仅本人」列表。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST/PUT `/api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/` 增 `repo_identities` | 工作空间成员 | Workspace / Task | write | `hasWorkspaceAccess` + `canMutateTask` | ✅ 充分 | 身份 ID 不校验归属亦可（与评论 POST 一致）；后续可 OPT 校验 identity.user_id |
| GET `/api/git-identities/user/{userId}/` | 认证用户 | User | read | urlUserID 必须等于 auth user | ✅ 充分 | 插件/工作台只拉当前用户 |
| 【自动运行】评论写入 JSON | 任务创建者 | Task / Comment | write | 内部 `ensureAutoRunAtComment` 用创建 userID | ✅ 充分 | 不对外新 API |

## 建模

不新增角色。身份选择器只展示当前用户身份，避免 IDOR 选他人 identity（列表本身已按 user 过滤）。
