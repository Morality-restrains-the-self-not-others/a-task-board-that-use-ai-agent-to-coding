# 注册邀请码 — NFR

| 类别 | 级别 | 说明 |
|---|---|---|
| 安全 | L3 | 注册闸门、superuser 门禁、一码一用防刷 |
| 可用性 | L2 | 配额耗尽可预期错误；关闭时零摩擦 |
| 性能 | L2 | 申请/核销单行事务；列表分页默认 50 |
| 可观测 | L2 | 结构化日志含 trace；事件 ISSUED/REDEEMED/POLICY |
| 一致性 | L3 | 核销与用户创建同事务或补偿删除用户 |

## 结构热点

CRG hub 不可用；人工热点：`handleEmailRegister`、配额 COUNT(issued_day)。
