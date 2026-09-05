# 权限分析：顶层交付物排队自动执行调度节奏

- 日期：2026-07-19
- 设计文档：`docs/superpowers/specs/2026-07-19-top-deliverable-queued-auto-run-schedule-design.md`
- 状态：goal-mode 自动完成

## 端点权限矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| PATCH todos `{schedule_rhythm}` | workspace 可写成员 | Workspace/Task | write | JWT + workspace access（既有 todos） | ✅ | 额外：仅 `parent_task_id` 空允许写节奏，否则 400 |
| PATCH todos `{queued_auto_run}` | workspace 可写成员 | Workspace/Task | write | 同上 | ✅ | 入队时校验存在顶层祖先节奏（可 disabled） |
| GET todos `{top}/queued-auto-run/` | workspace 可读成员 | Workspace/Task | read | 同上 list/get | ✅ | 校验 top 属于该 workspace |
| Cloud start-vm `started_via` | 服务间 / 用户启服 | Workspace/Task | write | 既有 start-vm 鉴权 | ✅ | 客户端不可伪造占额度：queued_schedule 仅 Internal/TTS 代调可写 |
| 手动启服出队 | 启服操作者 | Task | write | start-vm 权限 | ✅ | 出队与启服同事务/同请求副作用 |

## 角色

无新角色。沿用 tenant/workspace 成员读写边界。

## IDOR / 越权

| 风险 | 缓解 |
|------|------|
| 伪造他租户 top_task_id 入队 | membership 写入时强制 task.tenant_id/workspace_id 一致 |
| 非顶层写 schedule_rhythm | 服务端拒绝 parent 非空 |
| 外部把 started_via=queued_schedule 绕开并发 | Cloud：非 internal 请求忽略/覆盖为 manual |

## 结论

权限模型可落在既有 todos + compute 鉴权上；补充顶层写约束与 `started_via` 信任边界即可。
