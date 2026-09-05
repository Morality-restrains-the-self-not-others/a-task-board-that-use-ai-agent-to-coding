# NFR 澄清：runAll 远程日志推送（A2）

> 价值流：`docs/superpowers/plans/2026-06-04-runall-remote-log-shipping-value-stream.md`  
> 默认等级：**L2 Standard**（dev 可观测性）

| 类别 | 等级 | 场景（刺激 → 响应） |
|------|------|---------------------|
| 性能 | L2 | runAll tee 后 60s 内 Loki `{job="runall"}` 可查到新行 |
| 可用性 | L2 | local promtail 容器 `restart: unless-stopped`；Loki 不可达时 Promtail 重试（默认 client 行为） |
| 可维护性 | L2 | pipeline 与 `promtail.yaml` 测试断言一致；`LOKI_PUSH_URL` 来自 docker-infra SSOT |
| 安全 | L1 | dev Loki 无鉴权；不暴露到公网（文档约束） |
| 可扩展性 | L1 | 单开发者 Mac→单远程 CPU；不做多副本 Promtail |

**领域模型影响**：`LokiPushEndpoint` 值对象从 `Observability.LokiURL` 派生；无强一致聚合要求。
