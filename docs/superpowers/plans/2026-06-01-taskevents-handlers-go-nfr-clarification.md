# taskEvents Handler Go 化 — NFR 澄清

| 类别 | 级别 | 说明 |
|------|------|------|
| 可用性 | L2 | 单事件进程 crash 不影响其他事件；runAll 独立启停 |
| 一致性 | L3（G2+） | USER_CREATED/COMPANY_CREATED SQLite 与 Django ORM 行为等价 |
| 性能 | L2 | 去掉 Django HTTP 往返；SSE Redis publish <100ms p99 |
| 可观测性 | L2 | 沿用 tracelog + OTEL service name per event |
| 安全 | L3（G2+） | SQLite 只读/写最小表；internal secret 不进 bin/config.yaml |
| 运维 | L2 | bin/{event}/ 含二进制+config；8020–8049 端口段 |
