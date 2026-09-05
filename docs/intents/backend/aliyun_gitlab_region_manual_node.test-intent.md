# 测试意图：购买 GitLab 可选阿里云区域并由人工建节点开通

## 对应功能意图

[aliyun_gitlab_region_manual_node.intent.md](./aliyun_gitlab_region_manual_node.intent.md)

## 可执行测试

| ID | 场景 | 文件 |
|----|------|------|
| T1 | seed / list 含阿里云 pending_node | `taskBill/src/gitlab_region_aliyun_catalog_test.go` |
| T2 | pending_node 跳过 Admin API | 同上 + `gitlab_region.go` |
| T3 | 支付 pending_node 发事件且 pending_admin | `taskBill/src/order_payment_aliyun_fulfillment_test.go` |
| T4 | PUT infra_status ready | `taskBill/src/gitlab_region_admin_infra_status_test.go` |
| T5 | OrderCreate optgroup 阿里云 + 提示 | `taskFE/app/src/views/OrderCreate.contract.test.js` |
| T6 | groupGitlabRegionsByProvider 纯函数 | `taskFE/app/src/utils/gitlabRegionSelect.test.js` |
