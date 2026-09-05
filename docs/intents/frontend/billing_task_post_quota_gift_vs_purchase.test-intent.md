# 测试意图：账单页任务帖配额展示赠送与购买

## 测试目标

证明账单首页在 quotas 返回拆分字段时展示赠送/购买，缺失字段时不展示拆分行。

## 测试分层

- 前端单元：`taskFE/app/src/views/BillingDashboard.quotaSplit.test.js`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | gifted=50，purchased=33，total=83 | 拆分行含「赠送 50 帖」「购买 33 帖」，合计含 83 |
| T2 | 仅有 `task_post_quota` | 无 `task-post-quota-split` 节点 |

## 数据与环境

jsdom + 模拟 `apiFetch` quotas。

## 通过标准

T1–T2 全绿；既有 gitlabRegions / statisticsCards 不回归。

## 业务意图 → 事件对照（测试）

纯前端展示，不断言 MQ。
