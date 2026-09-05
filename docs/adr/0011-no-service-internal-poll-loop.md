# ADR-0011: 业务服务禁止进程内轮询 / 循环

- **Status:** accepted
- **Date:** 2026-08-16
- **Author:** Trae AI 团队
- **Deciders:** Trae AI 团队

---

## Context

monorepo 内多个业务进程（taskBill、taskCloudService、taskTaskService 等）在 HTTP server 之外再挂 `time.NewTicker`，用进程内循环做对账、过期、回收、分账。与此同时，`taskEvents` 已提供独立的 **timer worker**（`transport: timer`）和 **Kafka intent 消费者**：定时或事件到达后，HTTP 调用表 owner 的一次性 API。

进程内轮询导致触发点与业务进程生命周期耦合、无法独立扩缩/探活，并与「业务意图 → 领域事件」模型冲突。前端同源问题是无触发的 `setInterval` 打云 API。

需要一条架构约束：**服务只被叫醒，不自己按钟点扫。**

## Decision

**We will** 规定任意业务服务都不得以轮询/循环模式运行：

1. **周期工作**只允许放在专门的定时服务（`taskEvents` timer worker，独立 runAll 进程），由它 HTTP 触发业务服务的一次性接口
2. **即时工作**只允许由外界触发：HTTP/RPC、Kafka 消息、Webhook、显式 CLI、运维入口
3. 业务服务 `main` **禁止**新增 `time.NewTicker` / `time.Tick` / sleep 扫表循环
4. 前端禁止无用户操作、无 SSE 的后台 API 轮询
5. 存量 ticker 列入门禁 allowlist，禁止复制；功能触及即迁出

## Alternatives Considered

### Alternative 1: 每个服务自己 ticker，配置化间隔

- **Pros:** 实现快，少一个进程
- **Cons:** 每个服务重复时钟；失败与 HTTP 健康混在一起；无法按意图做消费者隔离
- **Why rejected:** 触发点必须可独立运维；与已有 taskEvents timer 形态重复且更差

### Alternative 2: 只用 Kafka 延时消息，不要 timer 进程

- **Pros:** 纯事件驱动
- **Cons:** 过期扫描、对账等「没有上游事件」的工作仍需要一个时钟源；Kafka 延时能力与运维复杂度更高
- **Why rejected:** 时钟源收敛到少数 timer worker 即可；不禁止未来用延时消息替代个别 timer

### Alternative 3: 仅文档禁止、不设 CI

- **Pros:** 改动面小
- **Cons:** Agent/开发者会继续在 `main` 里 `go ticker`
- **Why rejected:** 与「禁止忽略」元规则目标不符

## Consequences

### Positive

- 业务进程保持「请求进来才干活」
- 定时任务可独立重启、看 health、调间隔
- 与意图/Kafka 消费者模型一致

### Negative / Trade-offs

- 新增周期能力要多一个 runAll 条目和一次 internal API
- 存量 8 个业务 ticker 需后续迁出（门禁 allowlist）

### Mitigations

- 复制 `taskpostexpiryscan` 模板：timer 进程 + owner 一次性 API
- CI 阻断新 ticker；allowlist 只减不增

## References

- `.ai/01_project_constraints/51_no_service_internal_poll_loop.md`
- `.cursor/rules/no-service-internal-poll-loop.mdc`
- `docs/intents/frontend/comment_runtime_no_background_poll.intent.md`
- `taskEvents/internal/handlers/taskpostexpiryscan/runner.go`
