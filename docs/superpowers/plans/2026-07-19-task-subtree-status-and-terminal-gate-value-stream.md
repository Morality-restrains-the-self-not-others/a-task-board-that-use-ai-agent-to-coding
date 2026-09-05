# 价值流：任务子树状态与终态门禁

## 主价值流（增量）

```
打开任务详情 → 拉取 subtree → 展示子/孙状态与 settled/total
     ↓
尝试将父任务标为已完成/已取消 → DescendantTerminalGate
     ├─ 有开放后代 → 409 提示 → 用户先关闭子树
     └─ 全部 settled → 写库 → TASK_STATUS_CHANGED → 释放/SSE
```

## 测试点映射

| 价值流节点 | 测试点 | 用例 |
|------------|--------|------|
| 拉取 subtree | 深度与摘要 | T1 |
| 门禁拒绝 | 409 | T2 |
| 门禁通过 | 200 + 事件 | T3/T6 |
| 无子树 | 隐藏面板 / 通过 | T4 |
| 错误展示 | data-traceId | T5 |

## NFR（预告，详见 nfr 文档）

- L2：门禁查询在同 workspace 索引 `parent_task_id`；列名一次拉取缓存于请求内。
