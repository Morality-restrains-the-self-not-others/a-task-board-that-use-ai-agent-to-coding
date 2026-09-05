# NFR 澄清: task-events MQ 断连退避

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-03-task-events-mq-reconnect-backoff-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-03-task-events-mq-reconnect-backoff-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | MQ 断连时单进程 CPU < 1% |
| 可用性 | L2 | 运行中断连退避重连；启动 5 次 Ping 后 fail-fast |
| 容错机制 | L3 | 指数退避 + 限速日志 |
| 可观测性 | L2 | readiness 503 + mq_connected 字段 |
| 可伸缩性 | L0 | 不适用 |
| 安全性 | L0 | 不适用 |
| 数据一致性 | L0 | 不适用（broker 层） |

## 质量场景

### QS-01: MQ 断连 CPU 控制
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | Redis 停止 / 隧道断开 |
| 刺激 | 18 个 task-events 进程已启动并 Subscribe |
| 制品 | `broker/redis.go` consume loop |
| 环境 | 开发机，MQ 突然不可达 |
| 响应 | 各进程进入退避，无 tight loop |
| 响应度量 | `ps`/`top` 单进程 CPU < 1% 持续 60s |

### QS-02: 启动 fail-fast
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | runAll / run.sh start |
| 刺激 | Redis 未监听时启动 intent |
| 制品 | `broker.PingWithRetry` |
| 环境 | docker-redis 未启动 |
| 响应 | 5 次退避后 exit 1，日志含 ping failed |
| 响应度量 | 进程退出码 1；总等待 ≤ 35s |

### QS-03: Readiness 降级
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | MQ 运行中断连 |
| 刺激 | GET `/api/health/ready` |
| 制品 | `server/health.go` |
| 环境 | 进程存活，MQ 不可达 |
| 响应 | HTTP 503, status=degraded |
| 响应度量 | runAll readiness=degraded |

## 领域模型影响

| NFR 决策 | 模型影响 | DDD 动作 |
|----------|---------|---------|
| 退避 L3 | 连接策略属基础设施 | `BackoffPolicy` 值对象 |
| readiness L2 | 健康状态与 MQ 分离 | `ConnectionHealth` 值对象 |
| fail-fast L2 | 启动与运行策略不同 | 两套 BackoffPolicy 默认值 |

## 权衡与边界

### 取舍
- 启动 fail-fast 换取快速反馈，而非无限等待 MQ
- readiness 503 不 kill 进程，允许 MQ 恢复后自动继续

### 明确不做什么
- 不实现 circuit breaker 半开探测（退避足够）
- 不改 runAll 停服 kill 逻辑

### 升级触发条件
- 若 MQ 长时间不可用需自动 exit → 增加 max reconnect duration

## 跳过声明
- 可伸缩性/安全性/数据一致性：broker 基础设施修复，无新业务数据流
