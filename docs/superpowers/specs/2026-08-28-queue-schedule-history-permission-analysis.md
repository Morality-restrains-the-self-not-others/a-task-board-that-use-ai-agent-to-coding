# 自动调度安排 · 调度历史 — 权限分析

- 日期：2026-08-28
- 设计：`docs/superpowers/specs/2026-08-28-queue-schedule-history-design.md`

## 端点鉴权矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/queue-schedule/`（含 recent_history） | 租户成员且有工作空间访问 | Workspace | read | `hasWorkspaceAccess` | ✅ | 历史与快照同权 |
| GET `/queue-schedule/history/` | 同上 | Workspace | read | 同一 handler 入口 | ✅ 复用 | 禁止只校验 tenant |
| PUT `/queue-schedule/` | 同上 | Workspace | write | 既有 | ✅ | 本增量不放宽写 |
| append 内部 | 系统（dispatch/enqueue） | Workspace | write | 无公网 | ✅ | 不新增 internal HTTP |

无新角色、无新权限码。Viewer 可读历史（与读快照一致）。

## IDOR

- 路径含 tenant + workspace；SQL 必带 `workspace_id=?`
- cursor 只作为本 workspace 内排序键，不得用 id 跨空间取行
- 任务标题 JOIN 不得泄漏其他 workspace 任务

## 内部写

append fail-open。actor_user_id 仅 HTTP 用户路径填写；timer 为空串。
