# NFR Clarification: internal_dispatch Request 修复

> 价值流：`docs/superpowers/plans/2026-05-30-task-agent-support-internal-dispatch-request-fix-value-stream.md`

| 类别 | 等级 | 说明 |
|------|------|------|
| 可靠性 | L2 | exchange-refresh 不得因转发层缺陷 100% 失败；503 用于瞬时 DB busy |
| 可观测性 | L2 | 保留 `logger.exception` + `error_code`；Grafana 可区分 INTERNAL_DISPATCH_ERROR vs TOKEN_* |
| 性能 | L1 | 无新 I/O；RequestFactory 开销可忽略 |
| 安全 | L2 | internal secret 校验不变；不扩大 inbound 面 |
| 兼容性 | L2 | 对外 JSON 形状不变；仅消除错误 500 类 |

**领域影响：** 无新聚合；`TaskAgentSupportInternalDispatch` 属应用层适配，不进入领域层 ORM。
