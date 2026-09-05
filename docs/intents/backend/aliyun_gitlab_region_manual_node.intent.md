# 功能意图：购买 GitLab 可选阿里云区域并由人工建节点开通

## 意图

VIP1 在「购买资源」页选择 GitLab 区域时，除已部署的腾讯云实例外，必须能选择阿里云地域（目录先行、节点可尚未部署）。支付后该区配额为 `pending_admin`；平台运维在阿里云创建服务节点、挂载磁盘、登记 API/token 并将 `infra_status` 置 `ready` 后，再为租户开通 GitLab 组。禁止对 `pending_node` 区域调用 GitLab Admin API。

## 范围与边界

- 范围内：`billing_gitlab_region.infra_status` seed 阿里云区；OrderCreate 按云厂商分组；支付后事件；Admin PUT ready；跳过 auto-provision
- 范围外：自动创建阿里云 ECS；每租户独立 GitLab；OIDC/边缘 nginx 配方（节点就绪后运维另做）

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|----------|----------------|------------|--------|--------------|---------|
| 支付含 pending_node GitLab 行 | GitlabManualNodeFulfillmentQueued | gitlab-manual-node-fulfillment-queued | markOrderPaid | 审计/告警 | — |
| 运维标记节点就绪 | GitlabRegionInfraMarkedReady | gitlab-region-infra-marked-ready | PUT gitlab-regions | 审计 | — |
| GET 可售区域 | — | — | — | — | 纯查询 |

## 验收

- 购买页区域下拉含阿里云 optgroup，至少含 `aliyun-cn-hangzhou`
- 选中 pending_node 显示人工开通提示
- POST items.region 为阿里云 slug
- `ensureTenantGitlabGroupForRegion` 对 pending_node 不发 HTTP
- 支付后 `provisioning_status=pending_admin` 且发出 GitlabManualNodeFulfillmentQueued（key=order_id）
- SystemAdmin 区域卡可见「待创建节点」；PUT infra_status=ready 后可走既有开通
