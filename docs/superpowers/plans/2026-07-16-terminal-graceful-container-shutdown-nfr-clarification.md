# NFR：terminal-graceful-container-shutdown

- 日期：2026-07-16
- 默认等级：L2 Standard

| 类别 | 等级 | 说明 |
|------|------|------|
| 可用性 | L2 | 优雅失败回退硬释放；超时 90s 内最终释放 |
| 性能 | L2 | notify HTTP ≤5s；await 轮询间隔 ~5s |
| 安全 | L3 | 容器令牌边界；禁止跨任务释放 |
| 可观测性 | L2 | 结构化日志：notify/release/sole/timeout |
| 幂等 | L2 | shutdown/release/hard-release 均可重复 |
| 一致性 | L2 | 最终一致；Kafka 发布失败不回滚任务状态 |

## 领域模型影响

- Cloud Runtime：新增 MachineReleaseRequest、SoleContainerGate、GracefulShutdownAwait
- 无新聚合根；扩展 CSC 运行时状态语义
