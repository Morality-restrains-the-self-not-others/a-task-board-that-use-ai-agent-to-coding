# NFR 澄清：工作空间机器节点闲置策略

- 日期：2026-07-13
- 默认级别：L2 Standard（鉴权相关门禁按 L3）

| 质量属性 | 级别 | 场景 | 度量 | 对领域模型影响 |
|----------|------|------|------|----------------|
| 安全性 | L3 | 跨租户不得读写他租户 policy | path tenant 校验；403 | company_id 为 PK 部分 |
| 可用性 | L2 | summary/policy GET 失败不崩看板 | 前端降级显示「—」 | — |
| 性能 | L2 | summary 与 indicators 同级；15s 轮询 | 单 workspace SQL < 100ms 量级 | 索引 workspace_id 已有 |
| 一致性 | L2 | recycle 与终态释放竞态 | 幂等 stop；先到先得 | idle_since 更新原子 |
| 可观测性 | L2 | recycle/reuse 必有结构化日志 | event=idle_recycle / idle_reuse | — |
| 可扩展性 | L1 | 单 SQLite 扫描全 policy | MVP 可接受；日后按租户分片 | — |

## 明确不支持（本迭代）

- 跨 region 闲置复用 SLO
- 多容器共驻同一 VM 的资源隔离 NFR
