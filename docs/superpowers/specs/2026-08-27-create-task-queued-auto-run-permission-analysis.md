# 角色权限分析 — 创建任务可选加入自动调度队列

- 日期：2026-08-27
- 设计：`docs/superpowers/specs/2026-08-27-create-task-queued-auto-run-design.md`

无新端点、无新角色。沿用创建任务与 queue-schedule 既有鉴权。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/tenant/{tid}/workspace/{wid}/queue-schedule/` | 租户成员 | Workspace | read | `requireTenantMember` + `hasWorkspaceAccess` | ✅ 充分 | 前端只读一次 |
| POST/PUT todos `queued_auto_run` | 可创建/编辑该任务的成员 | Workspace / Task | write | 创建：租户+工作空间；更新：`hasWorkspaceAccess` + `canMutateTaskFromRequest` | ✅ 充分 | 不放宽入队 |
| 跳过立即 start-vm | 同上 | Task | write | 与 auto_run 同路径 | ✅ 充分 | queued 只推迟启服，不绕过 auto_run 门禁 |

IDOR：路径仍带 tenant_id + workspace_id；不根据用户输入改写 workspace。
无新角色/权限粒度。
