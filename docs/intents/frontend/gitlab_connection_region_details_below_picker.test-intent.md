# 测试意图：GitLab 连接页区域详情在下拉之下

## 测试目标

证明 GitLab 连接页区域选择器不被详情替代；详情在下拉之下，并随选项切换更新。

## 测试分层

- 前端单元：`taskFE/app/src/views/WorkspaceSettingsGitlabConnection.test.js`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `not_purchased` 且未选区域 | picker 可见；`gitlab-builtin-resources` 不存在 |
| T2 | `active` 且已选区域 | picker 与 details 同时存在；details 在 picker 内且位于 `gitlab-region-select` 之后 |
| T3 | `pending_admin` | picker 仍可见；等待开通文案在 details 内 |
| T4 | 下拉从区域 A 改为区域 B | details 展示 B 的名称；picker 仍在 |
| T5 | `not_purchased` 但已选区域 | picker 与 details 同时存在（空配额/未购买，不替换下拉） |

## 数据与环境

jsdom + mock `useGitlabResourcePurchase`；`availableRegions` 至少含两个 slug。

## 通过标准

T1–T5 全绿；无租户现金余额行 / 单价无「锁价」/ 连接 API 路径用例不回归。

## 业务意图 → 事件对照（测试）

纯前端展示，不断言 MQ。
