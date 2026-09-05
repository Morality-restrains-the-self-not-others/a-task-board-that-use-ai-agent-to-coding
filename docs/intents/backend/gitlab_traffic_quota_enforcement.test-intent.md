<!-- markdownlint-disable MD013 MD060 -->
# 测试意图：GitLab 出站流量配额强制执行

## 测试分层

- `taskBill/src/gitlab_traffic_gate_test.go`
- `taskBill/src/gitlab_traffic_usage_test.go`
- `taskBill/src/charge_no_wallet_test.go`
- `gitService/scripts/test_zzz_traffic_quota_initializer.sh`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | prepaid=0，公网 | allowed=false, TRAFFIC_NOT_PURCHASED |
| T2 | used≥prepaid>0 | allowed=false, TRAFFIC_QUOTA_EXCEEDED |
| T3 | used<prepaid | allowed=true |
| T4 | is_intranet（Host/VPC 内网，非 Docker NAT） | allowed=true / INTRANET_SKIP |
| T9 | 公网 Host + Docker NAT 源 IP | 不 skip，走配额 |
| T5 | project_path `tenant-{id}/repo` | 解析 tenant_id |
| T6 | 磁盘刷新 | 不抬高 traffic_used_gb |
| T7 | charge 预购不足 | 不累加 used |
| T8 | Ruby initializer | `ruby -c` 通过 |

## 通过标准

T1–T8 全绿。
