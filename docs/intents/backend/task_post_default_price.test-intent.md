# 测试意图：任务帖默认定价 0.55 元/帖/12个月

## 测试目标

证明全新库与缺行回退的任务帖单价均为 55 分（0.55 元），且上一版默认 33 分会被迁移更新。

## 测试分层

- 领域/单元：常量与分→元格式。
- 基础设施：`setupMySQLTestDB` 跑完全部 `dataMigrate/taskBill` 后读 `billing_unit`。
- 前端：价格管理 Playwright 夹具与定价页 mock 使用 0.55。

## 用例矩阵

| 编号 | 场景 | 期望 |
|------|------|------|
| T1 | `DefaultTaskPostUnitPriceCents` | 等于 55，`centsToYuanStr` 为 `0.55` |
| T2 | 迁移后读 `server_start` | `price=55`，`unit=帖/12个月` |
| T3 | `getCurrentResourcePricing` / `getUnitPriceCents(task_post)` | 返回 55 |
| T4 | 删除 `server_start` 行后再读单价 | 回退 55 |
| T5 | 公开定价缺行 | `task_points=55` |
| T6 | 前端价格管理 mock | 展示 `0.55` 元/帖 |

## 数据与环境

- Go：`TASKBILL_MYSQL_TEST_DSN` 或 registry 默认；`setupMySQLTestDB`。
- 前端：Vitest jsdom；Playwright 拦截 resource-pricing。

## 通过标准

T1–T5 对应 `taskBill/src/resource_pricing_default_test.go` 全绿；T6 对应 taskFE 既有测例夹具已改为 0.55。

## 业务意图 → 事件对照（测试）

本变更无新运行时事件。断言覆盖种子价与缺行回退，不要求 MQ 投递。
