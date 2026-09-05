# auto_run @ 评论 PR 回填 — NFR

| 类别 | 级别 | 说明 |
|------|------|------|
| 正确性 | L3 | 禁止二次 start-vm；PR 回填失败不回滚交付（soft-fail + 日志） |
| 幂等 | L2 | 评论复用 + auto_run_first/delivery 标志 |
| 性能 | L1 | 启服前多 1～2 次内部 HTTP；complete 一次出站 |
| 可观测 | L2 | `auto_run_at_comment_*` / `AUTO_RUN_PR_BACKFILL_*` 结构化日志 |
| 安全 | L2 | complete 仅容器 token；不经用户会话冒充 |

结构热点（CRG unavailable）：kickoff 分流与 delivery hooks。
