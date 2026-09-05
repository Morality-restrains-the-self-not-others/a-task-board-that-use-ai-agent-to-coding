# 下单施工留言（非自动发货资源）— 设计

- **Date:** 2026-08-19
- **Status:** accepted（goal-mode 自动采用）
- **Architecture change:** 否（无新服务 / 新事件 / 无 ArchiMate 升版）
- **ADR:** No-ADR: trivial tech choice, no architectural impact（既有 `billing_resource_order` 扩展可选字段）
- **python_api_approval:** not_applicable（全 Go + Vue）

## Goal / 成功标准

1. 创建订单时，若购物车含**需人工履约**资源（当前仅 `gitlab_disk`，支付后 `provisioning_status=pending_admin`），下单页展示可选留言框。
2. `POST /api/tenant/{tenant_id}/billing/orders/` 接受可选 `buyer_note`（trim 后 0～2000 字）；空串等价于不留言。
3. 仅自动发货资源（`task_post` / `gitlab_traffic`）却提交非空留言 → 400。
4. 留言持久化于 `billing_resource_order.buyer_note`；详情 JSON 回传；租户订单详情与超管订单展开可见（Vue 文本插值，防 XSS）。
5. 日志只记 `buyer_note_len`，禁止打正文。

## 现状

- 下单：`OrderCreate.vue` → `createOrder` → `billing_resource_order` + items。
- GitLab 磁盘支付后走 hybrid 开通，失败/待施工为 `pending_admin`；任务帖与流量为配额直发。
- 已有 `billing_order_comment` 是**成单后**租户↔超管线程，本增量是**下单瞬间**给施工人员的一次性说明，不双写评论表。

## 决策

| 方案 | 结论 |
|------|------|
| A. 订单主表 `buyer_note` | **采用**：一次写入、详情即可见、不依赖评论 UI |
| B. 创建时插入 `billing_order_comment` | 拒绝：成单后线程与施工备注语义不同；评论组件当前未挂详情页 |
| C. 行项级备注 | 拒绝：一期只有磁盘需施工，订单级足够 |

分类函数 SSOT：`resourceRequiresManualFulfillment(resourceType)`，当前仅 `gitlab_disk==true`。未来其它需施工 SKU 只改此函数。

## API

既有 `POST /api/tenant/{tid}/billing/orders/` body 增可选字段：

```json
{ "items": [...], "buyer_note": "请把组路径开到 team-foo" }
```

响应 `orderJSON` 增 `buyer_note`（缺省 `""`）。不新开端点。鉴权仍 `requireTenantAdmin`。超管读详情走既有租户订单 GET（staff 网关已放行）。

## DDL

`dataMigrate/taskBill/049_order_buyer_note.sql`：`billing_resource_order.buyer_note TEXT NOT NULL DEFAULT ''`。

冷热：订单主表非时间爆炸字段，不分区。

## 非目标

- 留言编辑/删除、附件、成单后自动进评论线程
- 改变 GitLab 开通状态机
- 新 Kafka 事件（下单成功路径仍用 `resource_order_created` 日志）
