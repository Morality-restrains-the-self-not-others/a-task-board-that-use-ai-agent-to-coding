# 价值流增量：入队前确认工作空间自动调度

## 最小可行增量

用户在任务详情要点「加入自动执行队列」→ 系统确认该工作空间已启用自动调度 → 才把任务加入队列；否则引导去「自动调度安排」。

```
点加入 → GET queue-schedule → enabled?
  ├─ 否 → 确认弹窗 → 前往设置 | 取消（不入队）
  └─ 是 → PATCH queued_auto_run=true → 离开队列按钮 + 状态 chip
```

## 测试点

- TP-JQ-1 已启用：GET + PATCH
- TP-JQ-2 未启用取消：无 PATCH
- TP-JQ-3 未启用确认：跳转 queue-schedule
- TP-JQ-4 GET 失败：data-traceId、无 PATCH
