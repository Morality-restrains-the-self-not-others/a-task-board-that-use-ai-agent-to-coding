# Review: edit-run auto delivery + PR comment

- **日期**: 2026-07-23
- **对照**: `2026-07-23-edit-run-auto-delivery-pr-comment-plan.md`

## 结论

无 critical / important 阻塞项。单测通过：Gateway edit-run、onlineServiceJS delivery/backfill、taskAIComment create auth、Cloud runtime-event allowlist。SPA 已 `npm run build` + collectstatic。

## Log / Intent 审计

| 项 | 结果 |
|----|------|
| 交付日志 | 复用 `AUTO_RUN_DELIVERY_*`；新增 `EDIT_RUN_AGENT_COMMENT_*`（onlineServiceJS + Cloud allowlist） |
| PR 回填 | `AUTO_RUN_PR_BACKFILL_*` + `kind=edit_run` 文案 |
| MQ 事件 | 证据豁免（与 006 一致，运行时事件非业务 MQ） |

## 残留风险

- 需滚动新 onlineServiceJS / Gateway / taskAIComment / Cloud 镜像或进程后公网才生效。
- 父评论缺少 `installed_image_id` 或已有活跃 Agent（409）时，交付仍执行但可能无法挂载评论回复。
