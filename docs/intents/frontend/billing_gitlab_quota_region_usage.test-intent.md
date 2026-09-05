# 测试意图：账单页 GitLab 按区用量

## 测试目标

账单资源配额区按区域渲染磁盘/流量已用与配额。

## 测试分层

- `taskFE/app/src/utils/formatUsedGb.test.js`
- `taskFE/app/src/views/BillingDashboard.gitlabRegions.test.js`
- `taskFE/app/src/views/BillingDashboard.gitlabQuotaSplit.test.js`
- `taskBill/src/gitlab_resources_test.go`（列表项含 used 字段）

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| F1 | 两区 + used | 两区名称；磁盘/流量文案含 `0.25 / 1` 与 `0.1 / 2`；无无区域汇总卡 |
| F2 | 赠送/购买 | 仍含「赠送 30 · 购买 70」 |
| F3 | 无 gitlab_resources | 不渲染按区块；回退卡含 `0.2 / 1` |
| F4 | formatUsedGb | `0` / 整数 / 去尾 0 |
| B1 | quotas 多区 | JSON 每项含 `disk_used_gb`、`traffic_used_gb` |

## 通过标准

相关 Vitest + Go 测例全绿。
