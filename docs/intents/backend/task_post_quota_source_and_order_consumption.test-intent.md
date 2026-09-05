# 测试意图：任务帖配额来源拆分与订单消耗归属

## 对应功能意图

`docs/intents/backend/task_post_quota_source_and_order_consumption.intent.md`

## 测试目标

证明赠送/购买剩余可查询；购买建批次；消耗归属订单并出现在订单 DTO；outbox payload 含来源字段。

## 测试分层

- 单元：taskBill MySQL fixture（`setupMySQLTestDB`）
- 前端：vitest mount BillingDashboard / BillingOrders / OrderDetail

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 仅赠送 10 | gifted=10 purchased=0 total=10 |
| T2 | 赠送 10 + 购买 5 | gifted=10 purchased=5 total=15 |
| T3 | markOrderPaid 任务帖 | grant source_kind=purchase order_id 匹配 |
| T4 | 有赠送时先消耗赠送 | 赠送 remaining-1；流水 related_order_id=赠送订单 |
| T5 | 赠送耗尽后消耗购买 | 购买批次 remaining-1；related_order_id=购买订单 |
| T6 | 订单 GET | resource_consumption.task_post.granted/consumed/remaining/events |
| T10 | GitLab 磁盘已支付订单 GET | resource_consumption.gitlab_disk.granted=行项数量，且不含 task_post 零值 |
| T11 | GitLab 磁盘有区域用量 | gitlab_disk.consumed/remaining 按该区域用量 LIFO 分摊到本单 |
| T12 | GitLab 流量已支付订单 GET | resource_consumption.gitlab_traffic.granted=行项数量 |
| T7 | 续存消耗 | action=renewal，仍归属批次订单 |
| T8 | 退款 | 该订单批次 remaining=0 |
| T9 | 消耗 outbox | payload 含 source_kind 与 order_id |
| F1 | 账单卡 | 可见「赠送」「购买」与数字 |
| F2 | 订单展开 | 可见「资源消耗」与已消耗/剩余 |
| F3 | 订单详情 | 同上 |

## 数据与环境

MySQL 测试库；前端 mock `apiFetch`。

## 通过标准

上表全部绿；既有 grant/订单单测回归通过。
