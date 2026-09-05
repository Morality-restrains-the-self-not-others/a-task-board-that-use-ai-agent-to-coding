# 测试意图：gitlab_disk_manual_admin_fulfillment

对应 `docs/intents/backend/gitlab_disk_manual_admin_fulfillment.intent.md`

## 测试目标

证明购买 GitLab 磁盘不会因 GET 配额自动建组；只有管理员开通实施才建组并发事件。

## 测试分层

- 领域/HTTP：`taskBill/src/gitlab_region.go` 视图、`order_payment.go`、`handlers` pending 列表
- 拟新增：`gitlab_disk_manual_fulfillment_test.go`（名称以实现为准）

## 用例矩阵

| ID | 场景 | 期望 |
|---|------|------|
| T1 | 支付 10GB 磁盘 | pending_admin，disk_gb>=10，无 GitLab HTTP |
| T2 | 支付后 GET regionResourceView 两次 | 不调用 ensure；状态仍 pending_admin |
| T3 | 支付发出 GitlabDiskFulfillmentQueued | topic 注册；payload 含 tenant_id/order_id/region；key=order_id |
| T4 | 超管 POST provision | active；ensure 被调用；GitlabTenantResourceProvisioned |
| T5 | 租户调 provision | 403 |
| T6 | GET pending-fulfillment 无超管 | 403 |
| T7 | GET pending-fulfillment 超管 | 含该租户该区 |
| T8 | 已 active 再支付加购 | 仍 active，disk_gb 增加 |
| T9 | pending_node 上 provision | 409 |
| T10 | 只买任务帖 | 立即加配额，不进 pending 列表 |

## 数据与环境

MySQL 测试库 + 替换 `ensureTenantGitlabGroupForRegion` / Kafka publisher 为桩。

## 通过标准

T1–T10 相关单测全绿。
