# 权限分析：任务子树状态与终态门禁

- 日期：2026-07-19
- 设计：`2026-07-19-task-subtree-status-and-terminal-gate-design.md`
- 结论：复用既有任务读写门禁；无新角色

## 端点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `.../todos/{id}/subtree/` | workspace 可读成员 | Workspace / Task | read | `hasWorkspaceAccess` + 任务租户归属 | ✅ 充分 | 与 GET task 一致 |
| PATCH `.../todos/{id}/`（终态） | 可写任务成员 | Task | write | `canMutateTaskFromRequest` | ✅ 充分 | 门禁在授权之后、写库之前 |
| PATCH `.../todos/{id}/switch` | 同上 | Task | write | workspace access | ✅ 充分 | completed→true 同样门禁 |
| progress-system columns 拉取 | TTS 服务侧 | Workspace | read | 服务间调用；失败降级 | ✅ | 不向客户端暴露额外权限面 |

## IDOR / 越权

- subtree 仅返回同 workspace 且以 path task 为根的后代；禁止跨租户。
- 门禁错误 payload 的 `open_descendants` 仅含本树节点 id/title/depth，无敏感字段。

## 状态约束

- 仅「进入终态」触发门禁；非终态列互转不检查子树。
- 无新 RBAC 角色。
