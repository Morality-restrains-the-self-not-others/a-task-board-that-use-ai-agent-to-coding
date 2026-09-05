# taskEvents Handler Go 化 — 价值流切片

> 输入：`2026-06-01-taskevents-handlers-relocation-design.md` v3

## 增量顺序（按价值与风险）

| 增量 | 事件二进制 | 价值 | 依赖 |
|------|-----------|------|------|
| I0 | G0 样板 + config | 后续事件可复制 | — |
| I1 | billing_transaction_created, sse_message | 低复杂度验证架构 | I0 |
| I1′ | email_sent, invitation_created, user_activated | notifications 拆进程 | I0 |
| I2 | user_created, company_created | 注册链核心 | I0 + SQLite repo |
| I3 | projects ×4 | 协作域 | I2 |
| I4 | cloud ×4 | 云资源 | ECS Go SDK |
| I5 | 删 Python handlers + 域级 cmd | 终态 | I1–I4 |

## 受影响 value-stream

- message-queue-kafka-to-redis：消费者 health 改为 per-event
- user-auth：I2 切换 user_created 二进制
- 云平台与资源：I4
