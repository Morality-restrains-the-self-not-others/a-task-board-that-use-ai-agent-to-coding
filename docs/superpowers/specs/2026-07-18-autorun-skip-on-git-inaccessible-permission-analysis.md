# 角色权限分析：Git 不可用时自动运行软跳过

**日期**: 2026-07-18  
**设计**: `docs/superpowers/specs/2026-07-18-autorun-skip-on-git-inaccessible-design.md`

## 结论

无新增权限点、无租户越权面变化。

| 主体 | 能力 | 边界 |
|------|------|------|
| 工作区成员（可创建/更新任务） | 保存 `auto_run=true` | 不变 |
| taskTaskService | 以请求用户 `user_id` 调用 internal nested-git-repos | 不得冒用他人 user_id |
| 云启动 | 仅在探测通过时由 schedule 触发 | 探测失败不得 start-vm |

## 威胁与缓解

| 风险 | 缓解 |
|------|------|
| 伪造 user_id 探测他人绑定 | 仍走网关鉴权后的 handler `userID`，不接受 body 覆盖 |
| 探测失败仍启动虚机浪费 | fail-closed 跳过 |
| 响应泄露 token | 仅回传 error 文案，不回传 token |

## 变更 endpoint

无新公开路由；仅 enrich 既有 create/update task 响应可选字段。
