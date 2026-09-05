# GitLab 磁盘价格套餐 — NFR 澄清

| 类别 | 级别 | 说明 |
|------|------|------|
| 可用性 | L2 | 沿用既有管理页与 API |
| 性能 | L2 | 宽表加列，无额外 N+1 |
| 安全/鉴权 | L3 | 超管写；字段为非负整数积分 |
| 可审计 | L2 | 套餐创建时间戳既有；无 PII |
| 一致性 | L3 | 套餐价与账户锁价同事务语义（换套餐一次 UPDATE） |
| 可观测 | L2 | 沿用 HTTP 访问日志；无密钥 |

对领域模型影响：PricingPackage / BillingAccount 各增一属性；BillingUnit 增一种 unit_type。
