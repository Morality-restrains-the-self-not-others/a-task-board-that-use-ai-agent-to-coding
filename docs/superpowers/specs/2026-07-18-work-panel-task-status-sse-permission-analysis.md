# 角色权限分析 — Work Panel 任务状态 SSE

日期：2026-07-18  
设计：`2026-07-18-work-panel-task-status-sse-design.md`

## 新/改 Endpoint

| 路径 | 方法 | Owner | 鉴权 |
|------|------|-------|------|
| `/api/tenant/{t}/workspace/{w}/work-panel-events-sse/` | GET (SSE) | taskSSE | Gateway token + internal secret；用户须能访问该租户/工作区（与 todos 同级 ACL） |

## 权限矩阵

| 角色 | 订阅本 workspace SSE | 收到他任务状态 | 伪造 user_id |
|------|----------------------|----------------|--------------|
| 租户成员（有 workspace 访问） | ✅ | ✅（同 workspace 可见任务状态字段） | ❌（忽略 query，信 X-User-Id） |
| 无租户/workspace 权限 | ❌ 网关拒 | — | — |
| 匿名 | ❌ | — | — |
| 内部无 secret 直连 taskSSE | ❌ 403 | — | — |

## 数据暴露

推送字段仅：`task_id`、`progress_column_id`、`completed`、列名等公开看板字段；无 AK/SK、无机器密码。

## 审计要点

- 连接建立/断开打结构化日志（含 tenant、workspace、user、traceId）
- fan-out 失败可重试，不回滚任务写库
