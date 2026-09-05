# 自动调度安排 · 排队任务卡可管理 — 权限分析

- 日期：2026-08-27
- 设计：`docs/superpowers/specs/2026-08-27-queue-schedule-members-manage-design.md`

## 端点鉴权矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/tenant/{tid}/workspace/{wid}/queue-schedule/` | 租户成员且有工作空间访问 | Workspace | read | `requireTenantMember` + `hasWorkspaceAccess` | ✅ 充分 | 本增量只读已加载快照 |
| PATCH `/api/tenant/{tid}/workspace/{wid}/todos/{taskId}/` `{queued_auto_run}` | 同上 | Task（属 workspace） | write | 既有 todos PATCH（租户成员 + workspace 归属 + 任务存在） | ✅ 充分 | 前端必须带 tid+wid+taskId，禁止只传 taskId |
| GET `/api/tasks/search/tenant_id/{tid}/` | 租户成员 | Tenant 任务搜索 | read | 既有 search（租户隔离）；query 带 `workspace_id` | ✅ 充分 | 前端再过滤 workspace + 已入队，防串空间 |

无新角色、无新权限码、无新端点。内部 dispatch / auto-close 不变。

## IDOR

- 入队/出队 URL 含 `tenant` + `workspace` + `taskId`；后端拒绝跨空间任务。
- 搜索结果若含其他 workspace，前端丢弃，不以搜索命中绕过 PATCH 路径上的 workspace。

## 前端

- 页面已要求选中工作空间；无 `workspaceId` 不渲染本卡。
- 403 沿用既有 `schedule-error` + `data-traceId`。
