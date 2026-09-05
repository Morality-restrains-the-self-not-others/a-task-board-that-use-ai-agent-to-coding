# NFR Clarification — tenant-gitlab-settings-resource-purchase

默认 L2（Standard）；计费路径关键项 L3。

| 类别 | 级别 | 要求 |
|------|------|------|
| 正确性/幂等 | L3 | 扣费事务 + 可选 idempotency_key；余额原子扣减 |
| 安全 | L2 | 登录必需；secret 脱敏；日志禁密钥 |
| 可观测 | L2 | 结构化日志 level 小写；错误带 traceId |
| 性能 | L2 | 单次购买 < 2s（本机 SQLite） |
| 可用性 | L2 | taskBill 不可用时代理 503 |
| 合规 | L2 | 积分消费，无支付卡数据 |
