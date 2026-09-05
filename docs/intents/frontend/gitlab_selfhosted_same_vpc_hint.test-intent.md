# 测试意图：自建 GitLab 同专有网络创建提示

## 测试目标

证明有默认机器节点时自建 GitLab 区块展示同 VPC 创建提示；无默认机器时不展示。

## 测试分层

- 前端单元：`taskFE/app/src/utils/defaultMachineNetworkHint.test.js`
- 前端单元：`taskFE/app/src/views/GitlabSelfHostedSameVpcHint.test.js`
- 前端单元：`taskFE/app/src/views/WorkspaceSettingsGitlabConnection.test.js`（显隐集成）

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 默认配置空数组 | `shouldShowSameVpcHint` 为 false |
| T2 | 至少一条默认配置 | hint 为 true；优先带 vpc 的条目为 primary |
| T3 | 页面 mock 空默认配置 | `gitlab-same-vpc-hint` 不存在 |
| T4 | 页面 mock 含 vpc/vswitch | hint 可见且含 VPC/交换机 ID |
| T5 | 无 vpc_id | 创建交换机按钮 disabled；创建 VPC 可用 |
| T6 | GET 失败 | 错误节点带 `data-traceId`；成功态 hint 不展示 |

## 通过标准

T1–T6 全绿；既有 GitLab 连接页用例不回归。

## 业务意图 → 事件对照（测试）

纯前端展示 + 复用既有创建弹窗，无新领域事件断言。
