<!-- markdownlint-disable MD013 MD060 -->
# 测试意图：GitLab 连接页已用流量

## 测试目标

证明设置页已用流量来自计量水位/扣费流水，**不**被磁盘占用持续抬高；
预购为 0 时分母为「未预购」，且 `traffic_download_allowed=false`。

## 测试分层

- 后端：`taskBill/src/gitlab_traffic_usage_test.go`
- 前端：`useGitlabResourcePurchase.test.js` /
  `WorkspaceSettingsGitlabConnection.test.js`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 已用磁盘有值、计量为 0 | `traffic_used_gb` 不为磁盘下限抬高；`traffic_download_allowed=false`（预购 0） |
| T2 | 扣费 GB > 计量 | 取扣费 GB |
| T3 | 计量已有值 | 不被磁盘下限覆盖 |
| T4 | floor 上报更小字节 | 计数不下降 |
| T5 | 设置页 GET 无 region，返回 resources[] | 再请求 `?region=` 并映射 traffic_used_gb |
| T6 | currentTrafficGb=0，trafficUsedGb=0.01049 | 展示含 0.01049 GB 与「未预购」（真实出站计量） |
| T7 | diskUsedGb=0.01049，trafficUsedGb=0 | 「已用磁盘」与「已用流量」数字不同 |

## 数据与环境

Go `setupMySQLTestDB`；前端 jsdom + mock `apiFetch`。

## 通过标准

T1–T6 全绿。

## 业务意图 → 事件对照（测试）

纯查询展示，不断言 MQ。
