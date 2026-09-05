# 测试意图：tenant_gitlab_resource_purchase

## 测试目标

证明租户 GitLab 资源购买不走 `billing_account.balance` 扣钱包，
只经资源订单支付发放配额。

## 测试分层

- 后端：`taskBill/src/gitlab_resources_http_purchase_test.go`
- 后端：`taskBill/src/charge_no_wallet_test.go`
- 前端：`taskFE/app/src/composables/useGitlabResourcePurchase.test.js`
- 前端：`taskFE/app/src/views/WorkspaceSettingsGitlabConnection.test.js`

## 用例矩阵

| ID | 场景 | 期望 |
|---|------|------|
| T1 | GET 无记录 | disk_gb=0, traffic_prepaid_gb=0，含锁价字段；无 `balance_points` |
| T2 | 资源订单支付 GitLab 磁盘/流量 | `markOrderPaid` 增加额度（磁盘可为 pending_admin），不减少 `balance` |
| T3 | POST `/purchase/` 且账户有余额 | 409 `USE_RESOURCE_ORDER`；余额不变；不发配额 |
| T4 | 侧栏文案 | 「GitLab」 |
| T5 | 页面购买入口 | `gitlab-purchase-link` 指向 `/billing/orders/create/` |
| T6 | 上报磁盘 + GET | disk_used_gb/traffic_used_gb 正确 |
| T7 | 设置页展示已用 | data-testid `gitlab-disk-used-gb` / `gitlab-traffic-used-gb` |
| T8 | 流量计量 | 不扣 `billing_account.balance` |
| T9 | 前端 composable | 不导出 `purchase`，不 POST `/purchase/` |

## 通过标准

T1–T9 相关单测全绿。
