# 价值流 — 创建任务可选加入自动调度队列

- 日期：2026-08-27
- 增量：create-task-queued-auto-run

## 最小可行增量

用户在工作面板创建任务并勾选自动运行时，若工作空间已启用自动调度，可选择入队而非立即启服。

```
打开创建任务 → GET queue-schedule
  → 勾选自动运行
  → [enabled] 显示加入队列勾选
      → 勾选提交：入队 + 跳过立即 start-vm → dispatcher 时段内 QueuedAutoRunStarted
      → 不勾选提交：立即 auto_run start-vm（现网）
  → [未启用] 无队列勾选，立即 auto_run
```

## 测试点

| ID | 步骤 | 断言 |
|----|------|------|
| TP-CTQ-1 | enabled + auto_run | 显示队列勾选，默认未勾 |
| TP-CTQ-2 | 未启用 | 不显示 |
| TP-CTQ-3 | 勾选提交 | `queued_auto_run=true`，无立即 start-vm |
| TP-CTQ-4 | 不勾选 | 立即 start-vm |
| TP-CTQ-5 | GET 失败 | 隐藏勾选 + data-traceId |

价值流图：`docs/flows/value-stream-test-integration.wsd` 矩形 `CTQAR`。
