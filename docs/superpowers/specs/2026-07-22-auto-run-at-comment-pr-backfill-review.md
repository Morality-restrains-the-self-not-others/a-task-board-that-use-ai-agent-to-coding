# auto_run @ 评论 PR 回填 — Review

- **日期**: 2026-07-22
- **设计**: `2026-07-22-auto-run-at-comment-pr-backfill-design.md`

## 核对

| 项 | 结果 |
|----|------|
| S1 合成 @ + Agent | ✅ `ensureAutoRunAtComment` + Go 单测 |
| S2 无 mention 启服事件 | ✅ 不调用 `publishDomainEvent` |
| S3 kickoff source=auto_run | ✅ postBootstrap 单测 |
| S4 PR 回填 | ✅ delivery hooks + backfill 单测 |
| 日志 | ✅ `auto_run_at_comment_*` / `AUTO_RUN_PR_BACKFILL_*` |
| Intent→Event | ✅ 证据豁免写明（防二次 start-vm） |

## CRG 风险

- graph_status: unavailable
- 手工风险：kickoff 分流错误会导致用户 @ 走错路径 — 已用 `source===auto_run` 单测锁住

## Log Audit

- TTS：`event=auto_run_at_comment_*`
- 容器：`AUTO_RUN_PR_BACKFILL_OK|FAILED` 已加入 runtime-event allowlist
