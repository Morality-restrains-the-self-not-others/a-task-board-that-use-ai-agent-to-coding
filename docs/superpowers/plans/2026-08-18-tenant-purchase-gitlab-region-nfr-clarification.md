# NFR 澄清 — 租户购买 GitLab 须按行选区

- **日期**: 2026-08-18
- **价值流**: `docs/superpowers/plans/2026-08-18-tenant-purchase-gitlab-region-value-stream.md`
- **默认等级**: L2；支付/配额路径按 L3 审视幂等（本增量不改支付幂等，只补下单校验）

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 可伸缩性 | 动作 |
|------|---------|----------|----------|------|
| `POST /api/tenant/{tid}/billing/orders/` | `tid` | 是 | L1 | 保持 |
| `GET /api/billing/gitlab-regions/tenant_id/{tid}/` | `tid` | 列表本身全局配置 | L0：区域数十条 | 升级触发：>1 万区分页 |
| `/tenant/:tid/billing/orders/create/` | `:tid` | 是 | L1 | 保持 |
| `/tenant/:tid/billing/orders/:orderId/` | tid + orderId | orderId 非分片键；tid 是 | L1 | 查询带 tenant |
| GitLab 设置页 `<a href>` | tid | 是 | L0 导航 | — |

`region` 是二级隔离键，查询始终带 `tenant_id`。

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| POST orders | 插入 pending 订单 | 双击 | 一次「创建订单」点击 | 无（现行为允许两笔 pending） | 不改；支付侧 `markOrderPaid` 按 order_id 幂等 |
| GET regions | 无 | — | — | L0 | — |
| 订单详情 GET | 无 | — | — | L0 | — |
| 设置页链接 | 无 | — | — | L0 | — |

禁止用 `tenant_id` 作下单幂等键。资金发放在支付回调，本增量不改。

## 类别等级

| 类别 | 等级 | 说明 |
|------|------|------|
| 可伸缩性 | L1 | tenant_id |
| 数据一致性 | L3 | 下单事务；支付发放另一事务（既有） |
| 安全 | L2 | tenant_admin + slug 白名单 |
| 可观测性 | L2 | `resource_order_created` 含 region 列表 |

## 质量场景

1. 刺激：VIP1 买磁盘未选区。响应：前端拦截，无 POST。
2. 刺激：`region=no-such-region`。响应：400 `region not found`。
3. 刺激：磁盘 `tencent-sh-1`、流量 `tencent-shanghai-5`。响应：两行 region 各异。

## 领域模型影响

`orderItemInput.Region` 升级为必须解析的 `GitlabRegionSlug` 值对象（GitLab 类型）。
