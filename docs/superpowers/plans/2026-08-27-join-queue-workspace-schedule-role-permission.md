# 角色权限：入队前校验工作空间自动调度

无新角色、无新端点。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/tenant/{tid}/workspace/{wid}/queue-schedule/` | 租户成员且有该工作空间访问 | Workspace | read | `requireTenantMember` + `hasWorkspaceAccess` | ✅ 充分 | 前端仅在加入点击时复用 |
| PATCH `.../todos/{taskId}/` `queued_auto_run` | 同上 | Workspace / Task | write | 既有任务 PATCH 鉴权 | ✅ 充分 | 前端未启用时不再调用 |

IDOR：tenantId/workspaceId 来自当前任务详情路由，与既有 toggle 一致。
