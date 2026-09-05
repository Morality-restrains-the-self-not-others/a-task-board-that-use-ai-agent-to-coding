# 价值流 — 评论运行态推送同步

- **日期**: 2026-08-16
- **设计**: `docs/superpowers/specs/2026-08-16-comment-runtime-no-background-poll-design.md`

## Related Value Streams

- `task-management` / `comment-binding-recover-after-start-success`：启动成功后 binding 恢复。本增量把**直播**从 Describe 轮询改为 SSE apply，不改 binding 表结构。
- `relay-stop-refresh-status-ui`：停机后刷新 UI。本增量要求停机完成 SSE 带 `runtime_status=Stopped`，按钮仍可对账。

## Increments（单切片即可交付）

1. **Push live** — 容器 heartbeat / 启动成功 / 停机完成 SSE 写入该评论 snapshot，零自动 Describe。
2. **Button reconcile** — 「刷新状态」唯一 GET `server-runtime-status?comment_id=`。
3. **Delete UI timers** — runtime / startup / binding / 遗留看板 `setInterval`。

## YAML

已追加 `conf/value-stream.yaml` → `task-management.comment-runtime-push-sync`。
