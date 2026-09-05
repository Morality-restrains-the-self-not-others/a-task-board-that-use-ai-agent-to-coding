# NFR：租户退款申请

默认 L2；资金与审批相关项抬升 L3。

| 类别 | 等级 | 要求 |
|------|------|------|
| 安全/权限 | L3 | 租户管理员申请；超管审批；internal secret |
| 一致性 | L3 | 冻结/解冻/核销同库事务；支付退款失败则审批失败不核销（或标记失败可重试） |
| 可观测性 | L2 | slog + trace；事件 outbox |
| 性能 | L1 | 审批低频；列表分页即可 |
| 合规隐私 | L3 | 日志脱敏；无卡号 |
| 韧性 | L2 | 支付 API 超时错误返回；mock 模式可测 |

## 结构热点

taskBill `creditRecharge` / `recordConsumptionUsage` 为 hub；新增 refund 旁路勿扩大扣费锁范围过久（SQLite BEGIN IMMEDIATE）。
