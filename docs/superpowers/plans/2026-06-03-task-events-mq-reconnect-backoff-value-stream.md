# Value Stream: task-events MQ 断连退避

> Derived from design: `docs/superpowers/specs/2026-06-03-task-events-mq-reconnect-backoff-design.md`

## Value Summary

开发者与 runAll 在 Redis/Kafka 未就绪时启动 task-events 消费者，不再遭遇 CPU 空转；进程要么快速失败，要么退避重连并通过 readiness 暴露 MQ 状态。

## Related Value Streams

- **domain-events-consumer-split**: extension — 增强 intent 级 health 与 broker 可靠性
- **message-queue-kafka-to-redis**: extension — Redis transport 运行期断连行为

## End-to-End Flow

[runAll 启动 task-events] → [Ping MQ] → [失败 exit / 成功 Subscribe] → [MQ 断连退避] → [readiness degraded] → [MQ 恢复继续消费]

## Value Increments

### Increment 1: Broker 退避 + 启动 Ping（Thin Slice）
**Value to user:** MQ 不可达时 CPU 不再暴涨；启动失败快速退出  
**Scope:** `broker/retry.go`, `broker/ping.go`, `redis.go`, `kafka.go`, `runner.go`  
**Depends on:** nothing

### Increment 2: Readiness Health + runAll split probe
**Value to user:** runAll UI 可区分进程存活 vs MQ 连通  
**Scope:** `server/health.go`, `runAll.yaml` liveness/readiness  
**Depends on:** Increment 1

### Increment 3: 测试与价值流字段
**Value to user:** 回归保障；value-stream 追踪 readiness 字段  
**Scope:** unit tests, `value-stream.yaml` step  
**Depends on:** Increment 2
