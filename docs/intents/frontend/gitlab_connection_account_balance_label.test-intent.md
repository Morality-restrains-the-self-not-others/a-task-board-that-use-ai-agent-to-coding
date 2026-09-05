<!-- markdownlint-disable MD013 MD060 -->
# 测试意图：租户侧不展示可花现金余额

## 测试目标

证明 GitLab 连接页与账单首页不向租户展示可花现金
（`billing_account.balance` / 冻结现金）。

## 测试分层

- 前端单元：
  `taskFE/app/src/views/WorkspaceSettingsGitlabConnection.test.js`
  `taskFE/app/src/views/BillingDashboard.switchNote.test.js`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | GitLab 内建资源区 active | 无 `gitlab-balance-points`；dt 无「账户余额」；有磁盘/流量单价 |
| T2 | 账单首页 `frozen_balance=500` | 全文不含「冻结金额」 |

## 数据与环境

jsdom + mock composable / `apiFetch`。

## 通过标准

T1–T2 全绿；区域选择器 / 单价无「锁价」不回归。

## 业务意图 → 事件对照（测试）

纯前端展示，不断言 MQ。
