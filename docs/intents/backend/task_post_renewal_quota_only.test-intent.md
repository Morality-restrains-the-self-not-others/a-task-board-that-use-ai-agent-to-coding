# 测试意图：任务帖续存只扣配额、不另扣费

## 测试目标

证明续存只减少 `task_post_quota`，不减少账户余额，也不按续存单价记付费消费。

## 测试分层

- 基础设施：`setupMySQLTestDB` 上跑 `consumeTaskPostRenewal`。
- 前端：管理端文案不再出现「续存同价」。

## 用例矩阵

| 编号 | 场景 | 期望 |
|------|------|------|
| T1 | 配额 3、余额 9999，续存 | 配额 2、余额 9999、`cost_cents=0` |
| T2 | 续存流水 | `billing_unit.unit_type=task_post_quota`，`amount=0` |
| T3 | 配额 0 续存 | 失败，余额不变 |
| T4 | 公开定价 | `task_renewal_points=0` |

## 通过标准

`taskBill/src/task_post_renewal_test.go` 与公开定价测例全绿。

## 业务意图 → 事件对照（测试）

续存成功路径须能产生 TASK_POST_RENEWED（taskTaskService 既有）及 amount=0 的配额审计流水。
