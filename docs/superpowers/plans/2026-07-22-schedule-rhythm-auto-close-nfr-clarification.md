# NFR：排队调度自动关闭

| 类别 | 级别 | 说明 |
|------|------|------|
| 可用性 | L2 | ticker 30s 粒度；预告窗口约 5 分钟内至少命中一次 |
| 安全 | L2→L3 边界 | 释放仅 internal；槽位范围隔离 |
| 性能 | L2 | 每 top 槽位数通常 ≤ max_queued_machines |
| 可观测 | L2 | warn/release 事件 + tracelog forward stage |
| 韧性 | L2 | 容器不可达 → stop-vm 回退 |

## 结构热点

CRG hub 不可用；人工判定热点：`dispatchTopQueue`、Cloud proxy 前缀、容器 lifecycle。
