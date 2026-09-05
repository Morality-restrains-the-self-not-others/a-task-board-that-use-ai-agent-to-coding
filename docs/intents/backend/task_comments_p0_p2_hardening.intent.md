# 意图：任务评论上线风险 P0–P2 硬化

## 意图

修复任务详情评论区上线阻断与高优先级风险：详情聚合空洞、列表分页/预览、Agent 流式批写、ownership/空壳表、合成错误可见、ContextPack/备份可观测。不迁 NoSQL。

## 验收

1. `GET .../todos/{id}/` 返回人类 `comments`；AI/Agent 经内部转发或前端并行补拉后刷新可见三类评论。
2. 列表支持 `limit`/`cursor`；AI/Agent 默认 `assistant_preview`。
3. Agent `/stream` DB 写经 80ms/2KB 批写；complete/fail 强制 flush。
4. `task-task` ownership 无 `ai_task_comments`；空壳表可 DROP；`container_agent_comments` 登记在 task-ai-comment。
5. 部分源失败时 `comments_feed_errors` 展示 banner。
6. 存在 SQLite 评论库备份 runbook。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 详情聚合/分页/批写/ownership/备份文档 | — | — | — | — | 纯查询与持久化优化，无新业务事实；证据豁免：无 publish |
| 人类发评 / @镜像 | — | — | — | — | 本期不改既有发评语义与事件；证据豁免：范围外 |
| AI 回复完成 | — | — | — | — | 本期不改既有完成事件；证据豁免：范围外 |

## 变更记录

- 2026-07-15：goal-mode 落地 P0–P2。
