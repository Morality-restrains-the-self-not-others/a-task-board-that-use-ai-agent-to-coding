# 价值流：Fork 确认弹窗副本数量

- **日期**: 2026-08-23
- **设计**: `docs/superpowers/specs/2026-08-23-fork-copy-count-design.md`

## Related Value Streams

扩展既有 Fork 确认流（`docs/intents/frontend/task_detail_fork_auto_run_confirm`），不是全新价值流。旧路径：Fork → 确认 auto_run → POST 1 次 → 新标签。新路径在确认前增加数量，确认后 POST N 次。

## 端到端增量

用户打开任务详情 → Fork → 确认弹窗选择副本数量（默认 1，最多 99）及是否自动运行 → 顺序创建 N 个派生任务（每份独立幂等键）→ 打开第一份详情 → 工作面板出现 N 张新卡。

## 切片（垂直，按交付顺序）

1. `clampForkCopyCount` 纯函数（1–99）
2. 模态数量选择 + confirm payload
3. `forkTask` 循环 POST + Idempotency-Key + 只打开第一份
4. Header / useTaskDetail 透传与防重放
5. Playwright：数量 3 → 3 次 POST

## 测试点

见 `docs/intents/frontend/task_detail_fork_auto_run_confirm.test-intent.md` T9–T13。
索引图 `docs/flows/value-stream-test-integration.wsd` Fork 注记追加数量选择。
