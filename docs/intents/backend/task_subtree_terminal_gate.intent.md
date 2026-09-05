# task_subtree_terminal_gate（功能意图 · 后端）

## 背景与目标

taskTaskService 为任务表 owner。需提供子树只读视图，并在父任务进入完成/取消终态时校验全部后代已关闭。

## 范围

- `GET .../todos/{id}/subtree/?max_depth=`
- PATCH / switch 上的 `DescendantTerminalGate`
- 终态判定对齐 taskEvents `ResolveTerminalKind`

## 验收

见设计文档 S1–S5 与 `034_task_subtree_status.test-intent.md` T1–T4、T6。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|----------|----------------|--------|--------------|---------|
| 查询子树 | — | — | — | 纯查询 |
| 父任务终态成功 | TASK_STATUS_CHANGED | TTS | taskEvents | 既有 |
| 门禁拒绝 | — | — | — | 无状态变更 |
