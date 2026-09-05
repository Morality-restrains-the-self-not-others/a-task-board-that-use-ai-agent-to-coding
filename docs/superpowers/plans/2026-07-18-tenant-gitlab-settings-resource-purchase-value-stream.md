# Value Stream — tenant-gitlab-settings-resource-purchase

**设计：** `2026-07-18-tenant-gitlab-settings-resource-purchase-design.md`

## 主价值流

**名称：** `tenant-gitlab-settings-resource-purchase`

| # | 步骤 | 状态 | 测试点 | 数据 |
|---|------|------|--------|------|
| 1 | 打开设置 → 侧栏见「GitLab」 | active | 前端文案 | Sidebar |
| 2 | GET 加载内建配额、已用量与锁价 | active | Go TestGitlabResourcesGet / Usage | billing_tenant_gitlab_resource |
| 2b | 设置页展示已用磁盘/流量 | active | data-testid gitlab-*-used-gb | disk_used_gb, traffic_used_gb |
| 3 | POST 购买磁盘+流量预购并扣费 | active | Go TestGitlabResourcesPurchase | balance↓, 配额覆盖 |
| 4 | 余额不足拒绝 | active | Go TestGitlabResourcesInsufficient | 402/400 |
| 5 | 下半区保存自建连接 | active | 既有 TestTenantGitlabConnectionPut | tenant_gitlab_oauth_connections |

## YAML（测试意图摘要）

```yaml
- name: tenant-gitlab-settings-resource-purchase
  steps:
    - name: menu-label-gitlab
    - name: get-resources
      asserts: [disk_gb, traffic_prepaid_gb, locked prices]
    - name: purchase-full-charge
      asserts: [balance, disk_gb, traffic_prepaid_gb]
    - name: self-hosted-connection
      asserts: [configured]
```
