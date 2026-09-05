<!-- markdownlint-disable MD013 MD060 -->
# 测试意图：GitLab 已用流量计量下限

## 测试目标

证明已用流量来自计量水位/扣费流水，**不**被磁盘占用或历史磁盘下限水位抬高。

## 测试分层

`taskBill/src/gitlab_traffic_usage_test.go`
`taskBill/src/gitlab_traffic_meter_test.go`
`gitService/scripts/test_ship_gitlab_git_traffic.py`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | disk_used_bytes=11263830，traffic_used_gb=0，billed=0 | 返回 0（磁盘不是流量） |
| T2 | billed=2 > 计量 | 返回 2 |
| T3 | 计量 1.5，磁盘更大 | 返回 1.5 |
| T4 | floor 1024 再 512 | 不下降 |
| T5 | regionResourceView 种子磁盘占用、traffic_used_gb=0 | JSON traffic_used_gb = 0 |
| T6 | 存量 traffic_used_gb 等于磁盘换算 GB（<1） | GET `traffic_used_gb` = 0，且 ≠ `disk_used_gb` |
| T7 | 整数 1 GB 计量且磁盘恰好 1 GiB | 保留 1（可能为扣费增量） |
| T8 | 10MiB clone bytes + 平台 GitLab URL | traffic_used_gb > 0 且 < 1（不取整成 1GB） |
| T9 | 同 `wh:{correlation_id}` 重放 | 计数不双加 |
| T10 | github.com URL | 跳过，计数不变 |
| T11 | 不同 region URL | 两区 used 各自增加且不相等 |
| T12 | 仅 `project_path=tenant-{id}/…` 无 tenant_id | 归集到该租户并抬高 used |
| T13 | 缺 region 且无 repo_url | skipped missing_region，不写入 defaultGitlabRegion |
| T14 | workhorse access 解析 git-upload-pack | 得到 project_path + bytes；CI / VPC 10/8 不发计量 |

## 通过标准

T1–T14 全绿。
