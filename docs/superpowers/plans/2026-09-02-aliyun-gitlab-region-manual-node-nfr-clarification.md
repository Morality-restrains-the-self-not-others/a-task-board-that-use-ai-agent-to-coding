# NFR 澄清 — 阿里云 GitLab 区域人工建节点

- **日期**: 2026-09-02
- **价值流**: `docs/superpowers/plans/2026-09-02-aliyun-gitlab-region-manual-node-value-stream.md`
- **默认等级**: L2；配额/支付路径幂等按 L3 审视（本增量不改 markOrderPaid 资金幂等，只加事件与跳过 API）

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 可伸缩性 | 动作 |
|------|---------|----------|----------|------|
| `GET /api/billing/gitlab-regions/tenant_id/{tid}/` | tid | 列表是全局目录，tid 仅鉴权 | L0：区域数十条 | 升级触发：>1 万区分页 |
| `POST /api/tenant/{tid}/billing/orders/` | tid | 是 | L1 | 保持 |
| `PUT /api/system-admin/gitlab-regions/{slug}/` | 无 tenant | 全局配置 | L0：超管低频 | 升级触发：多区域运营中心再按 cloud 分片 |
| `/tenant/:tid/billing/orders/create/` | tid | 是 | L1 | 保持 |
| Kafka `GitlabManualNodeFulfillmentQueued` | key=`order_id` | 订单级，非 tenant_id | L2 | 禁止用 tenant_id 作消费键 |
| Kafka `GitlabRegionInfraMarkedReady` | key=`slug` | 区域级配置，低频 | L0 | 升级触发：区域变更 QPS 上升再评估 |

`region` 是二级隔离键；租户配额查询始终带 `tenant_id`。

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| GET regions | 无 | — | — | L0 纯查询 | — |
| POST orders | 写 pending 订单 | 双击 | 一次创建点击 | 前端 Idempotency-Key（既有 clickGuard） | 保持既有 |
| markOrderPaid + 事件 | 配额 + Kafka | 支付回调重放 | 同一 order_id | order 已 paid 短路；事件 key=order_id | 重放不重复发放；事件 at-least-once，消费按 order_id |
| ensure skip pending_node | 无出站写 | 刷新配额页 | — | 自然空操作 | 不打 GitLab |
| PUT infra_status=ready | 改目录状态 | 双击保存 | 该 slug 转为 ready | 状态转移 L1；并发以最后写为准 | 已 ready 再 PUT ready 空操作 |
| admin provision | 调 GitLab API | 双击开通 | (tenant_id, region) | 组 path 幂等创建 | pending_node → 409 不调 API |

禁止用 `tenant_id` 作 Kafka 消费幂等键。资金发放仍按 order_id（既有 L3）。

## 类别等级

| 类别 | 等级 | 说明 |
|------|------|------|
| 可伸缩性 | L1 | 租户路径带 tid；目录 L0 |
| 数据一致性 | L3 | 支付事务既有；infra_status 与开通门闩 |
| 安全 | L2 | token 不进租户 GET；超管才改 infra |
| 可观测性 | L2 | 事件 + slog `gitlab_manual_node_fulfillment_queued` |
| 容错 | L2 | pending_node 失败不会误打本机 GitLab |

## 质量场景

1. 刺激：VIP1 打开购买页。响应：阿里云 optgroup 可见 `aliyun-cn-hangzhou`。
2. 刺激：选阿里云杭州买 1GB 并支付。响应：配额 pending_admin；无 GitLab HTTP；Kafka 事件 order_id。
3. 刺激：刷新账单 GitLab 视图。响应：不调用空 API。
4. 刺激：超管在 pending_node 上点开通。响应：409，说明节点未就绪。
5. 刺激：PUT ready 后开通。响应：走既有 ensure 组。

## 领域模型影响

`GitlabRegion` 增加值对象 `InfraStatus`。开通领域服务在 pending_node 短路。事件与 order_id / slug 同粒度。
