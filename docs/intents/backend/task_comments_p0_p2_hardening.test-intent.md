# 测试意图：任务评论上线风险 P0–P2 硬化

对应：`task_comments_p0_p2_hardening.intent.md`

## 用例

1. Go：创建人类评论后 `GET todos/{id}` 的 `comments` 非空。
2. Go：AI 服务不可达时详情仍返回人类评论，并含 `comments_feed_errors`。
3. Go/AIComment：列表 `?limit=` 返回 `results`/`next_cursor`；默认 preview 截断。
4. Go/AIComment：Agent stream 多次 chunk 仅少量 DB flush；complete 后全文完整。
5. Vitest：`fetchTaskDetail` 并行合并三源；失败写入 `comments_feed_errors`。
6. 前端：banner `data-testid=comments-feed-errors-banner` 在有错误时可见（组件 props）。

## 变更记录

- 2026-07-15：初版。
