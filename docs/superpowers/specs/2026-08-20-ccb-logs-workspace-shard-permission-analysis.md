# 评论启动日志 workspace 分表 — 角色权限分析

- **Date:** 2026-08-20
- **Design:** `docs/superpowers/specs/2026-08-20-ccb-logs-workspace-shard-design.md`
- **New roles:** 无

## 结论

无新 endpoint、无新角色。分片键 `workspace_id` 必须来自已鉴权路由（路径或 `X-Workspace-Id`），禁止客户端指定任意表名。跨工作空间读日志构成 IDOR，须用请求上下文中的 workspace 选片，不得用 body 覆盖。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET comment-container-bindings + logs | workspace 成员 | Workspace | read | 网关鉴权 + tenant/workspace 路由 | ✅ 充分 | list 必须用路由 workspace 选片 |
| POST create binding（写 pending 日志） | workspace 成员 | Workspace | write | 同上 | ✅ | create 传入同一 workspace |
| SSE persist 调度日志 | 内部/云回调 | Workspace | write | 容器 token / 内部 | ⚠️ 原路径无 workspace 列 | 从 statusData 或 CSC 解析，失败则跳过写入 |
| 直接 SQL 表名 | 服务进程 | System | write | 无用户输入 | ✅ | `ccbLogTable` 仅 `%02d` |

## IDOR

查询 `company_id + task_id` 而不带 workspace 会扫错片或漏数据。改为 `workspace_id + company_id + task_id`。不得接受 `shard` / `table` 查询参数。
