# NFR 澄清：任务状态变更事件与终态释放服务器

- 日期：2026-07-13
- 默认级别：L2 Standard（auth/财务无关，不升 L3）

## 质量属性场景

| 类别 | 级别 | 场景 | 度量 |
|------|------|------|------|
| 一致性 | L2 | 任务已更新但事件发布失败 | 记 ERROR；任务不回滚；可人工补释放；后续可加 outbox |
| 幂等 | L2 | 同一终态事件重复消费 | 第二次释放时无 instance/已 clear → no-op success |
| 延迟 | L2 | 终态到开始释放 | 通常 < 30s（Kafka + consumer）；非实时 UI 阻塞 |
| 可用性 | L2 | Consumer 短暂失败 | Retryable 错误 `DispatchRetry`；永久错误记日志 |
| 安全 | L2 | 跨租户释放 | payload 边界校验；服务凭据 |
| 可观测 | L2 | 发布/消费/释放分支 | 结构化日志含 task_id、terminal_kind、trace_id |
| 性能 | L1 | 非终态事件 | 快速 ack，不查云 API |

## 对领域模型影响

- 事件与任务写库 **最终一致**（非同事务 outbox）本期可接受。
- `ReleaseServersOnTerminal` 须幂等。
- 终态判定以列名约定为主，不引入强一致共享配置缓存。
