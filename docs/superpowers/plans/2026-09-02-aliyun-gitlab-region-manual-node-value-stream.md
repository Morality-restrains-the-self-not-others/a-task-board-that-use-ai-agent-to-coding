# 价值流 — 阿里云 GitLab 区域人工建节点

- **日期**: 2026-09-02
- **设计**: `docs/superpowers/specs/2026-09-02-aliyun-gitlab-region-manual-node-design.md`

## Related Value Streams

- `2026-08-18-tenant-purchase-gitlab-region-value-stream.md` — 购买必须选区；本流**扩展**下拉与开通门闩，不替换下单 API。
- `2026-08-18-pluggable-multi-region`（设计）— ADR-0014 可插拔实例；本流增加「可售但未部署」状态。

## 增量（按价值排序，单增量交付）

### Increment 1 — 可选阿里云并排队人工建节点（本迭代全部）

用户能在购买页选阿里云地域、下单支付；系统不自动建机；运维能看见待建节点并在就绪后开通。

| Step | 参与者 | 系统 | 测试 |
|------|--------|------|------|
| 选阿里云区域 | VIP1 | OrderCreate optgroup | OrderCreate.contract.test.js |
| 创建订单 | VIP1 | POST orders region=aliyun-* | 既有 orders_gitlab_region + 合同测 |
| 支付发放 | 支付 | pending_admin + 事件 | order_payment_aliyun_fulfillment_test.go |
| 跳过空实例 API | 系统 | pending_node skip ensure | gitlab_region_aliyun_catalog_test.go |
| 标记节点就绪 | 超管 | PUT infra_status | gitlab_region_admin_infra_status_test.go |

## Fields

- `taskBill.billing_gitlab_region.infra_status`
- `taskBill.billing_gitlab_region.cloud_provider`
- `taskBill.billing_gitlab_region.slug`
- `taskBill.billing_tenant_gitlab_resource.provisioning_status`
- `taskBill.billing_resource_order_item.region`

## Status

本增量 `active`。不把「自动 ECS」列入 planned（明确不做）。
