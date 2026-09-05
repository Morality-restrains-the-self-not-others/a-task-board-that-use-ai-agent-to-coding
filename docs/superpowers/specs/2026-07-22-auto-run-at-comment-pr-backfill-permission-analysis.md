# auto_run @ 评论 PR 回填 — 角色权限分析

- **日期**: 2026-07-22
- **设计**: `2026-07-22-auto-run-at-comment-pr-backfill-design.md`
- **状态**: approved（goal-mode）

## 权限矩阵

| 能力 | 角色 | 资源 | 操作 | 校验 | 状态 |
|------|------|------|------|------|------|
| 系统创建 auto_run @ 评论 | TTS 编排（任务 owner 上下文） | comments + container_agent | write | `scheduleTaskAutoRun` 已在 create/update 鉴权之后；系统路径跳过 workspace at_mode 用户闸 | ✅ |
| 不发 mention 启服事件 | 系统 | Kafka | n/a | **禁止** `TASK_COMMENT_IMAGE_MENTIONED`（防二次 start-vm） | ✅ |
| 容器回填 PR | 容器 access_token | container_agent complete | write | token → tenant/workspace/task 匹配 + agent_comment_id 归属 | ✅ 既有 |
| 读评论 Feed | 任务成员 | comments / container_agent | read | 既有任务访问权 | ✅ |

## CRG 触点

- graph_status: unavailable；触点同上设计 🕸️ 节。
- 敏感：`X-Access-Token` / internal secret 日志脱敏。
